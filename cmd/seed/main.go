package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/config"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("❌ seeding failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Set a reasonable timeout for connection + seeding execution combined
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Initialize connection pool
	pool, err := pgxpool.New(ctx, config.ENVS.DATABASE_URL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	slog.Info("✅ database connected and ready to seed")

	// 3. Initialize queries & seeder
	queries := sqlc.New(pool)
	seeder := postgres.NewSeeder(pool, queries)

	// 4. Execute seeding
	if err := seeder.Run(ctx); err != nil {
		return fmt.Errorf("execute seeders: %w", err)
	}

	slog.Info("✅ database seeded successfully")
	return nil
}
