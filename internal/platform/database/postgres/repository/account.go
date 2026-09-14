package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/db"
)

type accountRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewAccountRepository(pool *pgxpool.Pool, queries *db.Queries) account.Repository {
	return &accountRepository{
		pool:    pool,
		queries: queries,
	}
}

func (r *accountRepository) Create(ctx context.Context, params account.CreateAccountParams) (*account.Account, error) {
	return nil, nil
}

func (r *accountRepository) GetByID(ctx context.Context, id string) (*account.Account, error) {
	return nil, nil
}

func (r *accountRepository) GetByOwnerTypeCurrency(ctx context.Context, ownerID, accountType, currency string) (*account.Account, error) {
	return nil, nil
}

func (r *accountRepository) List(ctx context.Context, params account.ListAccountsParams) ([]account.Account, error) {
	return nil, nil
}

// Balance & Audit queries
func (r *accountRepository) GetBalance(ctx context.Context, accountID string) (int64, error) {
	return 0, nil
}

func (r *accountRepository) GetStatement(ctx context.Context, accountID string) ([]account.StatementEntry, error) {
	return nil, nil
}

func (r *accountRepository) Reconcile(ctx context.Context, accountID string) (*account.ReconciliationResult, error) {
	return nil, nil
}
