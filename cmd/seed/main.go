// Command seed loads reference data: currencies, account types and system accounts.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/config"
	"github.com/mide7/go-financial-ledger-api/internal/logger"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

const serviceName = "seed"

func main() {
	os.Exit(start())
}

// start loads configuration and the logger, seeds, and returns the exit code.
// The logger is closed after the final error is logged.
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

	slog.SetDefault(log)

	code := 0

	if err := run(cfg, log); err != nil {
		log.Error("❌ seeding failed", "err", err)

		code = 1
	}

	if err := closeLog(); err != nil {
		fmt.Fprintln(os.Stderr, "close log outputs:", err)
	}

	return code
}

func run(cfg *config.Config, log *slog.Logger) error {
	// 1. Set a reasonable timeout for connection + seeding execution combined
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Initialize connection pool
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	// pgxpool connects lazily; ping so a bad DATABASE_URL fails here, not on first query.
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	log.InfoContext(ctx, "✅ database connected and ready to seed")

	// 3. Initialize queries & seeder
	queries := sqlc.New(pool)
	seeder := postgres.NewSeeder(pool, queries, log)

	// 4. Execute seeding
	if err := seeder.Run(ctx); err != nil {
		return fmt.Errorf("execute seeders: %w", err)
	}

	log.InfoContext(ctx, "✅ database seeded successfully")

	return nil
}
