package main

import (
	"context"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, config.ENVS.DATABASE_URL)
	if err != nil {
		slog.Error("failure while connecting to database via pgxpool: %v", err)
		os.Exit(1)
	}

	db := stdlib.OpenDBFromPool(pool)

	defer func() {
		if err := db.Close(); err != nil { // This is the ONLY close you need for the DB
			slog.Error("error while closing *sql.DB wrapper: %v", err)
		}
	}()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("failure while creating postgres driver: %v", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithDatabaseInstance("file://internal/migrations", "postgres", driver)
	if err != nil {
		slog.Error("failure while creating migrate instance: %v", err)
		os.Exit(1)
	}

	cmd := os.Args[(len(os.Args) - 1)]
	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			slog.Error("failure while running migration up: %v", err)
			os.Exit(1)
		}
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			slog.Error("failure while running migration down: %v", err)
			os.Exit(1)
		}
	}
}
