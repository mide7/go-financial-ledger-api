// Command migrate applies (up) or reverts (down) the database migrations.
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
	"github.com/mide7/go-financial-ledger-api/internal/logger"
)

// migrationsSource is relative to the working directory, so the command must run
// from the repository root (as `make migrate-up` and the dev container do).
const migrationsSource = "file://internal/platform/database/postgres/migrations"

// Supported commands.
const (
	cmdUp   = "up"
	cmdDown = "down"
)

// errUsage is returned for a missing, unknown or extra command-line argument.
var errUsage = errors.New("usage: migrate up|down")

const serviceName = "migrate"

func main() {
	os.Exit(start(os.Args[1:]))
}

// start validates the command, loads configuration and the logger, migrates,
// and returns the exit code. The command is checked before anything else, so
// a typo is reported even when the database or config is unavailable.
func start(args []string) int {
	cmd, err := parseCommand(args)
	if err != nil {
		slog.Error("❌ migration failed", "err", err)
		return 2
	}

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

	if err := run(cmd, cfg, log); err != nil {
		log.Error("❌ migration failed", "err", err)

		code = 1
	}

	if err := closeLog(); err != nil {
		fmt.Fprintln(os.Stderr, "close log outputs:", err)
	}

	return code
}

// parseCommand validates the arguments before anything touches the database.
func parseCommand(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%w: got %d arguments", errUsage, len(args))
	}

	switch args[0] {
	case cmdUp, cmdDown:
		return args[0], nil
	default:
		return "", fmt.Errorf("%w: unknown command %q", errUsage, args[0])
	}
}

// run applies cmd (already validated by parseCommand) to cfg.DatabaseURL.
func run(cmd string, cfg *config.Config, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgxpool connection failed: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("error closing database pool wrapper", "err", err)
		}
	}()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver instance: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsSource, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// Always close migrate instance to release advisory locks
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil || dbErr != nil {
			log.Error("error closing migrate instance", "srcErr", srcErr, "dbErr", dbErr)
		}
	}()

	return execute(m, cmd, log)
}

func execute(m *migrate.Migrate, cmd string, log *slog.Logger) error {
	switch cmd {
	case cmdUp:
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migration up failed: %w", err)
		}

		log.Info("✅ database migrations applied (up)")
	case cmdDown:
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migration down failed: %w", err)
		}

		log.Info("✅ database migrations reverted (down)")
	default:
		return fmt.Errorf("%w: unknown command %q", errUsage, cmd)
	}

	return nil
}
