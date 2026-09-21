package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mide7/go-financial-ledger-api/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("❌ migration failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Validate arguments explicitly
	if len(os.Args) < 2 {
		return fmt.Errorf("missing command argument: expected 'up' or 'down'")
	}
	cmd := os.Args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Connect pool
	pool, err := pgxpool.New(ctx, config.ENVS.DATABASE_URL)
	if err != nil {
		return fmt.Errorf("pgxpool connection failed: %w", err)
	}

	// 3. Wrap pool in *sql.DB standard interface
	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("error closing database pool wrapper", "err", err)
		}
	}()

	// 4. Initialize migration driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver instance: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/platform/database/postgres/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// Always close migrate instance to release advisory locks
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil || dbErr != nil {
			slog.Error("error closing migrate instance", "srcErr", srcErr, "dbErr", dbErr)
		}
	}()

	// 5. Execute command
	switch cmd {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migration up failed: %w", err)
		}
		slog.Info("✅ database migrations applied (up)")
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migration down failed: %w", err)
		}
		slog.Info("✅ database migrations reverted (down)")
	default:
		return fmt.Errorf("invalid command %q: expected 'up' or 'down'", cmd)
	}

	return nil
}
