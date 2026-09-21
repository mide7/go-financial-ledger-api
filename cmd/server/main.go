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
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/repository"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
	transportHttp "github.com/mide7/go-financial-ledger-api/internal/platform/transport/http"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handlers"
)

func main() {
	if err := run(); err != nil {
		slog.Error("❌ service shutdown due to error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Initialize DB Pool
	pool, err := pgxpool.New(ctx, config.ENVS.DATABASE_URL)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	slog.Info("✅ database connected")

	// 2. Build Dependency Tree
	queries := sqlc.New(pool)

	currencyRegistry, err := postgres.NewCurrencyRegistry(ctx, pool, queries)
	if err != nil {
		return fmt.Errorf("failed to create currency registry: %w", err)
	}

	accountRepository := repository.NewAccountRepository(pool, queries)
	transactionRepository := repository.NewTransactionRepository(pool, queries)

	accountService := account.NewAccountService(accountRepository, currencyRegistry)
	transactionService := transaction.NewTransactionService(transactionRepository, currencyRegistry)

	handler := handlers.NewHandler(accountService, transactionService)
	healthCheckHandler := handlers.NewHealthCheckHandler(pool)

	port := config.ENVS.PORT
	router := transportHttp.NewRouter(handler, healthCheckHandler)
	httpServer := transportHttp.NewHttpServer(port, router)

	// 3. Start Server with Error Channel
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("🚀 HTTP server starting", "port", port)
		if err := httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 4. Wait for OS Signal or Startup Failure
	select {
	case err := <-serverErr:
		return fmt.Errorf("HTTP server startup failed: %w", err)
	case <-ctx.Done():
		slog.Warn("⏳ OS signal received, initiating graceful shutdown sequence...")
	}

	// 5. Graceful Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("failed to gracefully shutdown HTTP server: %w", err)
	}

	slog.Warn("✅ graceful shutdown complete")
	return nil
}
