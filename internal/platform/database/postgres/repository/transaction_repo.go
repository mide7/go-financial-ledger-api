package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
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

	if err := lockPostableAccounts(ctx, qtx, params.Entries, account.Status.AllowsEntry); err != nil {
		return nil, err
	}

	created, err := insertTransaction(ctx, qtx, params)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

// lockPostableAccounts share-locks every account the entries post to, in id
// order, and checks each entry against its account's status with allows. The
// lock is held until the database transaction ends, so a concurrent status
// change (FOR UPDATE) waits for the posting instead of slipping in after the
// check. It returns ErrEntryAccountNotFound for an unknown account and
// ErrAccountNotPostable for the first entry its account's status refuses.
func lockPostableAccounts(
	ctx context.Context,
	qtx *sqlc.Queries,
	entries []transaction.EntryParams,
	allows func(account.Status, domain.EntryType) bool,
) error {
	ids := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.AccountID)
	}

	slices.SortFunc(ids, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	ids = slices.Compact(ids)

	rows, err := qtx.LockAccountsForPosting(ctx, ids)
	if err != nil {
		return err
	}

	statuses := make(map[uuid.UUID]account.Status, len(rows))
	for _, row := range rows {
		statuses[row.ID] = account.Status(row.Status)
	}

	for _, e := range entries {
		status, ok := statuses[e.AccountID]
		if !ok {
			return fmt.Errorf("%w: %s", transaction.ErrEntryAccountNotFound, e.AccountID)
		}

		if !allows(status, e.Type) {
			return fmt.Errorf("%w: account %s is %s, entry is %s",
				transaction.ErrAccountNotPostable, e.AccountID, status, e.Type)
		}
	}

	return nil
}

// insertTransaction inserts the header and its entries with qtx, which must be
// bound to a database transaction, and returns them as stored.
func insertTransaction(
	ctx context.Context,
	qtx *sqlc.Queries,
	params transaction.CreateTransactionParams,
) (*transaction.Transaction, error) {
	header, err := qtx.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		Reference:           params.Reference,
		Type:                params.Type,
		Currency:            params.Currency.String(),
		ParentTransactionID: params.ParentTransactionID,
		Description:         params.Description,
		Metadata:            params.Metadata,
		TransactionDate:     params.TransactionDate,
	})
	if err != nil {
		return nil, mapInsertTransactionError(err)
	}

	entryParams := make([]sqlc.CreateEntriesParams, len(params.Entries))
	for i, entry := range params.Entries {
		entryParams[i] = sqlc.CreateEntriesParams{
			TransactionID: header.ID,
			AccountID:     entry.AccountID,
			Type:          sqlc.EntryType(entry.Type),
			Amount:        entry.Amount,
		}
	}

	// The integrity trigger runs at the end of this statement.
	if _, err := qtx.CreateEntries(ctx, entryParams); err != nil {
		return nil, mapIntegrityError(err)
	}

	// CopyFrom returns only a row count; read the entries back for their ids
	// and sequences.
	entries, err := qtx.ListEntriesByTransactionID(ctx, header.ID)
	if err != nil {
		return nil, err
	}

	created := mapToTransactionDomain(&header)
	created.Entries = mapToEntryDomainList(entries)

	return created, nil
}

// mapInsertTransactionError turns the header insert's unique violations into
// domain errors: the one-reversal index means the parent is already reversed,
// any other means the reference is taken.
func mapInsertTransactionError(err error) error {
	switch {
	case isUniqueViolationOf(err, uniqueOneReversal):
		return transaction.ErrAlreadyReversed
	case isUniqueViolation(err):
		return transaction.ErrDuplicateReference
	default:
		return err
	}
}

// GetByID returns the transaction with its entries, in entry_sequence order.
func (r *transactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	row, err := r.queries.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, notFound(err, transaction.ErrTransactionNotFound)
	}

	// Entries are immutable and written with their header in one database
	// transaction, so a second read outside a transaction is consistent.
	entries, err := r.queries.ListEntriesByTransactionID(ctx, id)
	if err != nil {
		return nil, err
	}

	txn := mapToTransactionDomain(&row)
	txn.Entries = mapToEntryDomainList(entries)

	return txn, nil
}

// List returns transaction headers, newest first, without their entries.
func (r *transactionRepository) List(
	ctx context.Context,
	params transaction.ListTransactionsParams,
) ([]transaction.Transaction, error) {
	rows, err := r.queries.ListTransactions(ctx, sqlc.ListTransactionsParams{
		AccountID: params.AccountID,
		Reference: optionalText(params.Reference),
		Type:      optionalText(params.Type),
		Currency:  optionalText(params.Currency.String()),
		StartDate: optionalTimestamptz(params.StartDate),
		EndDate:   optionalTimestamptz(params.EndDate),
		Cursor:    params.Cursor,
		Limit:     params.Limit,
	})
	if err != nil {
		return nil, err
	}

	txns := make([]transaction.Transaction, len(rows))
	for i := range rows {
		txns[i] = *mapToTransactionDomain(&rows[i])
	}

	return txns, nil
}

// Reverse posts a REVERSAL of transaction params.ID in one database
// transaction: the original's entries with DEBIT and CREDIT swapped, in the
// original's currency, with parent_transaction_id pointing at the original
// and the reason as the description.
//
// The GetReversalOf check gives the common case a clear error; two concurrent
// reversals both pass it, and the uq_transactions_one_reversal index rejects
// the second insert with the same ErrAlreadyReversed.
func (r *transactionRepository) Reverse(
	ctx context.Context,
	params transaction.ReverseTransactionParams,
) (*transaction.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	original, err := qtx.GetTransactionByID(ctx, params.ID)
	if err != nil {
		return nil, notFound(err, transaction.ErrTransactionNotFound)
	}

	if original.Type == transaction.TypeReversal {
		return nil, transaction.ErrCannotReverseReversal
	}

	if _, err := qtx.GetReversalOf(ctx, uuid.NullUUID{UUID: original.ID, Valid: true}); err == nil {
		return nil, transaction.ErrAlreadyReversed
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	originalEntries, err := qtx.ListEntriesByTransactionID(ctx, original.ID)
	if err != nil {
		return nil, err
	}

	mirrored := make([]transaction.EntryParams, len(originalEntries))
	for i, e := range originalEntries {
		mirrored[i] = transaction.EntryParams{
			AccountID: e.AccountID,
			Type:      domain.EntryType(e.Type).Opposite(),
			Amount:    e.Amount,
		}
	}

	allowsReversal := func(s account.Status, _ domain.EntryType) bool { return s.AllowsReversal() }
	if err := lockPostableAccounts(ctx, qtx, mirrored, allowsReversal); err != nil {
		return nil, err
	}

	reversal, err := insertTransaction(ctx, qtx, transaction.CreateTransactionParams{
		Reference:           params.Reference,
		Type:                transaction.TypeReversal,
		Currency:            currency.Code(original.Currency),
		ParentTransactionID: uuid.NullUUID{UUID: original.ID, Valid: true},
		Description:         params.Reason,
		Metadata:            []byte("{}"),
		TransactionDate:     time.Now(),
		Entries:             mirrored,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return reversal, nil
}

// optionalText is a NULL text parameter when s is empty, so the query skips that filter.
func optionalText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// optionalTimestamptz is a NULL timestamptz parameter when t is the zero time.
func optionalTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
