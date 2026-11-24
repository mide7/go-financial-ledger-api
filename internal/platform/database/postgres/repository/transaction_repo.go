package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

type transactionRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewTransactionRepository returns a transaction.Repository backed by pool.
func NewTransactionRepository(pool *pgxpool.Pool, queries *sqlc.Queries) transaction.Repository {
	return &transactionRepository{
		pool:    pool,
		queries: queries,
	}
}

func (r *transactionRepository) Create(
	ctx context.Context,
	params transaction.CreateTransactionParams,
) (*transaction.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	createdTransaction, err := qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		Reference:           params.Reference,
		Type:                params.Type,
		Currency:            params.Currency.String(),
		ParentTransactionID: params.ParentTransactionID,
		Description:         params.Description,
		Metadata:            params.Metadata,
		TransactionDate:     params.TransactionDate,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, transaction.ErrDuplicateReference
		}

		return nil, err
	}

	entryParams := make([]sqlc.CreateEntriesParams, len(params.Entries))
	for i, entry := range params.Entries {
		entryParams[i] = sqlc.CreateEntriesParams{
			TransactionID: createdTransaction.ID,
			AccountID:     entry.AccountID,
			Type:          sqlc.EntryType(entry.Type),
			Amount:        entry.Amount,
		}
	}

	if _, err := qtx.CreateEntries(ctx, entryParams); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return mapToTransactionDomain(&createdTransaction), nil
}

func (r *transactionRepository) GetByID(_ context.Context, _ string) (*transaction.Transaction, error) {
	return nil, domain.ErrNotImplemented
}

func (r *transactionRepository) List(
	_ context.Context,
	_ transaction.ListTransactionsParams,
) ([]transaction.Transaction, error) {
	return nil, domain.ErrNotImplemented
}

func (r *transactionRepository) Reverse(_ context.Context, _ string) error {
	return domain.ErrNotImplemented
}
