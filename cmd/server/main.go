// Command server runs the ledger HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/config"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/domain/withdrawal"
	"github.com/mide7/go-financial-ledger-api/internal/logger"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/repository"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
	"github.com/mide7/go-financial-ledger-api/internal/platform/idempotency"
	httptransport "github.com/mide7/go-financial-ledger-api/internal/platform/transport/http"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

const (
	serviceName = "ledger-api"
	// idempotencyPurgeInterval is how often expired idempotency records are deleted.
	idempotencyPurgeInterval = time.Hour
)

func main() {
	os.Exit(start())
}

// start loads configuration, builds the logger and runs the server, returning
// the process exit code. The logger is closed only after the final error is
// logged, so file outputs receive it.
func start() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("❌ invalid configuration", "err", err)
		return 1
	}

	log, closeLog, err := logger.New(cfg.LoggerOptions(serviceName))
	if err != nil {
		slog.Error("❌ invalid logger configuration", "err", err)
		return 1
	}

	// Safety net for third-party code and any stray package-level slog call;
	// application code receives log explicitly.
	slog.SetDefault(log)

	code := 0

	if err := run(cfg, log); err != nil {
		log.Error("❌ service shutdown due to error", "err", err)

		code = 1
	}

	if err := closeLog(); err != nil {
		fmt.Fprintln(os.Stderr, "close log outputs:", err)
	}

	return code
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// pgxpool connects lazily; ping so a bad DATABASE_URL fails here, not on first query.
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	log.Info("✅ database connected")

	queries := sqlc.New(pool)

	currencyRegistry, err := repository.NewCurrencyRegistry(ctx, pool, queries, log)
	if err != nil {
		return fmt.Errorf("failed to create currency registry: %w", err)
	}

	accountRepository := repository.NewAccountRepository(pool, queries)
	transactionRepository := repository.NewTransactionRepository(pool, queries)

	accountService := account.NewAccountService(accountRepository, currencyRegistry, log)
	transactionService := transaction.NewTransactionService(transactionRepository, currencyRegistry)
	withdrawalService := withdrawal.NewWithdrawalService()

	apiHandler := handler.NewHandler(handler.Services{
		Accounts:     accountService,
		Transactions: transactionService,
		Withdrawals:  withdrawalService,
		Currencies:   currencyRegistry,
	}, log)
	healthCheckHandler := handler.NewHealthCheckHandler(pool)

	idempotencyStore := repository.NewIdempotencyStore(queries)
	idempotent := middleware.Idempotency(idempotencyStore, cfg.IdempotencyTTL, log)

	go purgeIdempotencyRecords(ctx, idempotencyStore, idempotencyPurgeInterval, log)

	router := httptransport.NewRouter(apiHandler, healthCheckHandler, idempotent)
	httpServer := httptransport.NewServer(cfg.Port, router, log)

	serverErr := make(chan error, 1)

	go func() {
		if err := httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("HTTP server startup failed: %w", err)
	case <-ctx.Done():
		log.Info("⏳ OS signal received, initiating graceful shutdown")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("failed to gracefully shutdown HTTP server: %w", err)
	}

	log.Info("✅ graceful shutdown complete")

	return nil
}

// purgeIdempotencyRecords deletes expired idempotency records every interval
// until ctx is done. A failed purge is logged and retried on the next tick;
// expired records are reclaimable anyway, so a missed purge only costs space.
func purgeIdempotencyRecords(
	ctx context.Context,
	store idempotency.Store,
	interval time.Duration,
	log *slog.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := store.DeleteExpired(ctx)
			if err != nil {
				log.WarnContext(ctx, "purge expired idempotency records", "err", err)
				continue
			}

			log.DebugContext(ctx, "purged expired idempotency records", "count", n)
		}
	}
}
