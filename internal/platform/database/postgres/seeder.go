package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

type Seeder struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewSeeder(pool *pgxpool.Pool, queries *sqlc.Queries) *Seeder {
	return &Seeder{
		pool:    pool,
		queries: queries,
	}
}

func (s *Seeder) Run(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	if err := s.seedCurrencies(ctx, qtx); err != nil {
		return fmt.Errorf("seed currencies: %w", err)
	}

	slog.Info("🌱 Seeding currencies...")
	if err := s.seedSystemAccounts(ctx, qtx); err != nil {
		return fmt.Errorf("seed system accounts: %w", err)
	}

	slog.Info("🌱 Seeding system accounts...")
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}

func (s *Seeder) seedCurrencies(ctx context.Context, qtx *sqlc.Queries) error {
	currencies := []sqlc.SeedCurrencyParams{
		{Code: "USD", Name: "US Dollar", Exponent: 2, IsActive: true},
		{Code: "EUR", Name: "Euro", Exponent: 2, IsActive: true},
		{Code: "NGN", Name: "Nigerian Naira", Exponent: 2, IsActive: true},
	}

	for _, c := range currencies {
		if err := qtx.SeedCurrency(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) seedSystemAccounts(ctx context.Context, qtx *sqlc.Queries) error {
	accounts := []sqlc.SeedSystemAccountParams{
		{OwnerID: "admin", Type: "USER_WALLET", Currency: "NGN"},
		{OwnerID: "admin", Type: "PLATFORM_ESCROW_LIABILITY", Currency: "NGN"},
		{OwnerID: "admin", Type: "PLATFORM_FEE_REVENUE", Currency: "NGN"},
		{OwnerID: "admin", Type: "MERCHANT_PAYABLE", Currency: "NGN"},
	}

	for _, a := range accounts {
		if err := qtx.SeedSystemAccount(ctx, a); err != nil {
			return err
		}
	}
	return nil
}
