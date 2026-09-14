package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/db"
)

type transactionRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewTransactionRepository(pool *pgxpool.Pool, queries *db.Queries) transaction.Repository {
	return &transactionRepository{
		pool:    pool,
		queries: queries,
	}
}

func (r *transactionRepository) Create(ctx context.Context, params transaction.CreateTransactionParams) (*transaction.Transaction, error) {
	return nil, nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id string) (*transaction.Transaction, error) {
	return nil, nil
}

func (r *transactionRepository) List(ctx context.Context, params transaction.ListTransactionsParams) ([]transaction.Transaction, error) {
	return nil, nil
}

func (r *transactionRepository) Reverse(ctx context.Context, id string) error {
	return nil
}
