// Package postgres holds PostgreSQL setup that sits outside the repositories, such as seeding.
package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

// Seed values repeated across the reference data below.
const (
	systemOwnerID   = "admin"
	systemCurrency  = "NGN"
	natureLiability = "LIABILITY"
	balanceCredit   = "CREDIT"
)

// Seeder inserts the reference data the ledger needs to run.
type Seeder struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	log     *slog.Logger
}

// NewSeeder returns a Seeder that writes through pool and logs through log.
func NewSeeder(pool *pgxpool.Pool, queries *sqlc.Queries, log *slog.Logger) *Seeder {
	return &Seeder{
		pool:    pool,
		queries: queries,
		log:     log,
	}
}

// Run seeds currencies, account types and system accounts in one transaction.
func (s *Seeder) Run(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := s.queries.WithTx(tx)

	s.log.InfoContext(ctx, "🌱 seeding currencies")

	if err := s.seedCurrencies(ctx, qtx); err != nil {
		return fmt.Errorf("seed currencies: %w", err)
	}

	s.log.InfoContext(ctx, "🌱 seeding account types")

	if err := s.seedAccountTypes(ctx, qtx); err != nil {
		return fmt.Errorf("seed account types: %w", err)
	}

	s.log.InfoContext(ctx, "🌱 seeding system accounts")

	if err := s.seedSystemAccounts(ctx, qtx); err != nil {
		return fmt.Errorf("seed system accounts: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}

func (s *Seeder) seedCurrencies(ctx context.Context, qtx *sqlc.Queries) error {
	currencies := []sqlc.SeedCurrencyParams{
		{Code: "USD", Name: "US Dollar", Exponent: 2, IsActive: true},
		{Code: "EUR", Name: "Euro", Exponent: 2, IsActive: true},
		{Code: systemCurrency, Name: "Nigerian Naira", Exponent: 2, IsActive: true},
	}

	for _, c := range currencies {
		if err := qtx.SeedCurrency(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

func (s *Seeder) seedAccountTypes(ctx context.Context, qtx *sqlc.Queries) error {
	accountTypes := []sqlc.SeedAccountTypeParams{
		{Type: "USER_WALLET", Nature: natureLiability, NormalBalance: balanceCredit, Description: "User spendable balance per currency"},
		{Type: "USER_LOCKED", Nature: natureLiability, NormalBalance: balanceCredit, Description: "User funds held pending withdrawal/dispute"},
		{Type: "MERCHANT_WALLET", Nature: natureLiability, NormalBalance: balanceCredit, Description: "Merchant available balance per currency"},
		{Type: "MERCHANT_PAYABLE", Nature: natureLiability, NormalBalance: balanceCredit, Description: "Settled merchant funds queued for payout"},
		{Type: "MERCHANT_RESERVE", Nature: natureLiability, NormalBalance: balanceCredit, Description: "Rolling reserve held against chargebacks/refunds"},
		{Type: "PLATFORM_ESCROW_LIABILITY", Nature: natureLiability, NormalBalance: balanceCredit, Description: "Funds held mid-transaction before release"},
		{Type: "PLATFORM_FEE_REVENUE", Nature: "REVENUE", NormalBalance: balanceCredit, Description: "Platform earned fees"},
		{Type: "PLATFORM_CASH", Nature: "ASSET", NormalBalance: "DEBIT", Description: "Real money held at bank/payment processor per currency"},
		{Type: "PLATFORM_FX_SUSPENSE", Nature: natureLiability, NormalBalance: balanceCredit, Description: "Clearing account bridging the two legs of a currency conversion"},
		{Type: "PLATFORM_FX_GAIN_LOSS", Nature: "REVENUE", NormalBalance: balanceCredit, Description: "Realized FX spread/slippage from conversions"},
	}
	for _, a := range accountTypes {
		if err := qtx.SeedAccountType(ctx, a); err != nil {
			return err
		}
	}

	return nil
}

// seedSystemAccounts creates one system-owned account per account type, so the
// list can neither repeat nor miss a type as types are added.
func (s *Seeder) seedSystemAccounts(ctx context.Context, qtx *sqlc.Queries) error {
	for _, accountType := range account.AllTypes {
		if err := qtx.SeedSystemAccount(ctx, sqlc.SeedSystemAccountParams{
			OwnerID:  systemOwnerID,
			Type:     string(accountType),
			Currency: systemCurrency,
		}); err != nil {
			return fmt.Errorf("%s: %w", accountType, err)
		}
	}

	return nil
}
