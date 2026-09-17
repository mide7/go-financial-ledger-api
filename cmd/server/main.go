package main

import (
	"context"
	"errors"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, config.ENVS.DATABASE_URL)
	if err != nil {
		slog.Error("failed to create database connection pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		slog.Error("unable to ping database", "err", err)
		os.Exit(1)
	}
	slog.Info("✅ database connection pool ping successful")

	queries := sqlc.New(pool)

	currencyRegistry, err := postgres.NewCurrencyRegistry(ctx, pool, queries)
	if err != nil {
		slog.Error("failed to create currency registry", "err", err)
		os.Exit(1)
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

	go func() {
		if err = httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	slog.Warn("⏳ initiating graceful shutdown sequence...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = httpServer.Stop(shutdownCtx); err != nil {
		slog.Error("failed to gracefully shutdown HTTP server", "err", err)
	}

	slog.Warn("✅ graceful shutdown complete")
}
