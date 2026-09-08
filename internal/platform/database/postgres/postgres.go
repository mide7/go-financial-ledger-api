package postgres

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/config"
)

func NewPostgresStorage(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, config.ENVS.DATABASE_URL)
	if err != nil {
		slog.Error("unable to create connection pool: %v\n", err)
	}

	return pool, nil
}

func InitStorage(ctx context.Context, pool *pgxpool.Pool) error {
	err := pool.Ping(ctx)
	if err != nil {
		slog.Error("unable to ping database: %v\n", err)
		return err
	}

	slog.Warn("✅ connected to database")
	return nil
}

func CloseStorage(pool *pgxpool.Pool) {
	if pool != nil {
		slog.Warn("⏳ closing database connection pool...")
		pool.Close()
		slog.Warn("✅ database connection pool closed")
		return
	}

	slog.Warn("no active database connection pool")
}
