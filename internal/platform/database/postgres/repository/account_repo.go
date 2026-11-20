// Package repository implements the domain repositories and currency registry on PostgreSQL.
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

type accountRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewAccountRepository returns an account.Repository backed by pool.
func NewAccountRepository(pool *pgxpool.Pool, queries *sqlc.Queries) account.Repository {
	return &accountRepository{
		pool:    pool,
		queries: queries,
	}
}

func (r *accountRepository) Create(ctx context.Context, params account.CreateAccountParams) (*account.Account, error) {
	createdAccount, err := r.queries.CreateAccount(ctx, sqlc.CreateAccountParams{
		OwnerID:  params.OwnerID,
		Type:     params.Type,
		Currency: params.Currency,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, account.ErrAccountExists
		}

		return nil, err
	}

	return mapToAccountDomain(&createdAccount), nil
}

func (r *accountRepository) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	row, err := r.queries.GetAccountByID(ctx, id)
	if err != nil {
		return nil, notFound(err, account.ErrAccountNotFound)
	}

	return mapToAccountDomain(&row), nil
}

func (r *accountRepository) GetByOwnerTypeCurrency(ctx context.Context, params account.GetAccountByOwnerTypeCurrencyParams) (*account.Account, error) {
	row, err := r.queries.GetAccountByOwnerTypeCurrency(ctx, sqlc.GetAccountByOwnerTypeCurrencyParams{
		OwnerID:  params.OwnerID,
		Type:     params.Type,
		Currency: params.Currency,
	})
	if err != nil {
		return nil, notFound(err, account.ErrAccountNotFound)
	}

	return mapToAccountDomain(&row), nil
}

// List returns one page of accounts. params.Page is 1-based; values below 1 are
// treated as the first page.
func (r *accountRepository) List(ctx context.Context, params account.ListAccountsParams) ([]account.Account, error) {
	offset, err := pageOffset(params.Page, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("offset: %w", err)
	}

	limit, err := toInt32(params.Limit)
	if err != nil {
		return nil, fmt.Errorf("limit: %w", err)
	}

	accounts, err := r.queries.ListAccounts(ctx, sqlc.ListAccountsParams{
		OwnerID:   pgtype.Text{String: params.OwnerID, Valid: params.OwnerID != ""},
		Type:      pgtype.Text{String: params.Type, Valid: params.Type != ""},
		Currency:  pgtype.Text{String: params.Currency, Valid: params.Currency != ""},
		SortBy:    pgtype.Text{String: params.SortBy, Valid: params.SortBy != ""},
		SortOrder: pgtype.Text{String: params.OrderBy, Valid: params.OrderBy != ""},
		Offset:    offset,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	return mapToAccountDomainList(accounts), nil
}

// Balance & Audit queries.
func (r *accountRepository) GetBalance(ctx context.Context, accountID uuid.UUID) (*account.GetAccountBalanceResult, error) {
	balance, err := r.queries.GetAccountBalance(ctx, accountID)
	if err != nil {
		return nil, notFound(err, account.ErrAccountNotFound)
	}

	// Formatted in Go from exact minor units; dividing in SQL goes through floating point.
	return &account.GetAccountBalanceResult{
		Currency:     balance.Currency,
		Exponent:     balance.Exponent,
		BalanceMinor: balance.BalanceMinor,
		BalanceMajor: currency.FormatMinor(balance.BalanceMinor, int(balance.Exponent)),
	}, nil
}

func (r *accountRepository) GetStatementSummary(ctx context.Context, params account.GetStatementSummaryParams) (*account.StatementSummary, error) {
	summary, err := r.queries.GetAccountStatementSummary(ctx, sqlc.GetAccountStatementSummaryParams{
		AccountID: params.AccountID,
		StartDate: pgtype.Timestamptz{Time: params.StartDate, Valid: true},
		EndDate:   pgtype.Timestamptz{Time: params.EndDate, Valid: true},
	})
	if err != nil {
		return nil, notFound(err, account.ErrAccountNotFound)
	}

	return &account.StatementSummary{
		AccountID:      summary.AccountID,
		Currency:       currency.Code(summary.Currency),
		Exponent:       summary.Exponent,
		OpeningBalance: summary.OpeningBalance,
		ClosingBalance: summary.ClosingBalance,
		TotalDebits:    summary.TotalDebits,
		TotalCredits:   summary.TotalCredits,
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
	}, nil
}

func (r *accountRepository) ListStatementEntries(ctx context.Context, params account.GetStatementEntriesParams) ([]account.StatementEntry, error) {
	rows, err := r.queries.ListAccountStatementEntries(ctx, sqlc.ListAccountStatementEntriesParams{
		AccountID: params.AccountID,
		StartDate: params.StartDate,
		EndDate:   params.EndDate,
		Cursor:    params.Cursor,
		Limit:     params.Limit,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]account.StatementEntry, len(rows))

	for i, entry := range rows {
		entryType, err := domain.ParseEntryType(string(entry.Type))
		if err != nil {
			return nil, fmt.Errorf("corrupt data for entry %s: %w", entry.ID, err)
		}

		entries[i] = account.StatementEntry{
			ID:             entry.ID,
			TransactionID:  entry.TransactionID,
			Type:           entryType,
			Amount:         entry.Amount,
			RunningBalance: entry.RunningBalance,
			Currency:       currency.Code(entry.Currency),
			Exponent:       entry.Exponent,
			Description:    entry.Description,
			EntrySequence:  entry.EntrySequence,
			CreatedAt:      entry.CreatedAt,
		}
	}

	return entries, nil
}

func (r *accountRepository) Reconcile(ctx context.Context, accountID uuid.UUID) (*account.ReconciliationResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	lockedAccount, err := qtx.GetAccountForUpdate(ctx, accountID)
	if err != nil {
		return nil, notFound(err, account.ErrAccountNotFound)
	}

	result, err := qtx.ReconcileAccount(ctx, lockedAccount.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &account.ReconciliationResult{
		AccountID:            result.AccountID,
		Status:               result.Status,
		Currency:             currency.Code(result.Currency),
		Exponent:             result.Exponent,
		RawLedgerSum:         result.RawLedgerSum,
		SnapshotBalance:      result.SnapshotBalance,
		DeltaSinceSnapshot:   result.DeltaSinceSnapshot,
		CalculatedTotal:      result.CalculatedTotal,
		DiscrepancyAmount:    result.DiscrepancyAmount,
		LastSnapshotSequence: result.LastSnapshotSequence,
		CurrentMaxSequence:   result.CurrentMaxSequence,
		ReconciledAt:         result.ReconciledAt,
	}, nil
}

func (r *accountRepository) ChangeAccountStatus(ctx context.Context, params account.StatusChangeParams) (*account.StatusHistory, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	lockedAccount, err := qtx.GetAccountForUpdate(ctx, params.AccountID)
	if err != nil {
		return nil, notFound(err, account.ErrAccountNotFound)
	}

	if params.ExpectedStatus != nil && *params.ExpectedStatus != account.Status(lockedAccount.Status) {
		return nil, fmt.Errorf("account id %q is not in expected status %q", params.AccountID, *params.ExpectedStatus)
	}

	if err = qtx.UpdateAccountStatus(ctx, sqlc.UpdateAccountStatusParams{
		AccountID: params.AccountID,
		Status:    params.NewStatus.String(),
	}); err != nil {
		return nil, err
	}

	result, err := qtx.InsertStatusHistory(ctx, sqlc.InsertStatusHistoryParams{
		AccountID: params.AccountID,
		Reason:    params.Reason,
		ActorType: params.ActorType.String(),
		ActorID:   params.ActorID,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &account.StatusHistory{
		ID:             result.ID,
		AccountID:      result.AccountID,
		PreviousStatus: mapStringPointerToAccountStatusPointer(result.PreviousStatus),
		NewStatus:      account.Status(result.NewStatus),
		Reason:         result.Reason,
		ActorType:      account.ActorType(result.ActorType),
		ActorID:        result.ActorID,
		CreatedAt:      result.CreatedAt,
	}, nil
}

func (r *accountRepository) GetAccountStatus(ctx context.Context, accountID uuid.UUID) (account.Status, error) {
	status, err := r.queries.GetAccountStatus(ctx, accountID)
	if err != nil {
		return "", notFound(err, account.ErrAccountNotFound)
	}

	return account.Status(status), nil
}

func (r *accountRepository) ListAccountStatusHistory(ctx context.Context, params account.ListAccountStatusHistoryParams) ([]account.StatusHistory, error) {
	var cursorCreatedAt pgtype.Timestamptz
	if params.CursorCreatedAt != nil {
		cursorCreatedAt = pgtype.Timestamptz{Time: *params.CursorCreatedAt, Valid: true}
	}

	rows, err := r.queries.ListAccountStatusHistory(ctx, sqlc.ListAccountStatusHistoryParams{
		AccountID:       params.AccountID,
		CursorID:        params.CursorID,
		CursorCreatedAt: cursorCreatedAt,
		Limit:           params.Limit,
	})
	if err != nil {
		return nil, err
	}

	histories := make([]account.StatusHistory, len(rows))
	for i, row := range rows {
		histories[i] = account.StatusHistory{
			ID:             row.ID,
			AccountID:      row.AccountID,
			PreviousStatus: mapStringPointerToAccountStatusPointer(row.PreviousStatus),
			NewStatus:      account.Status(row.NewStatus),
			Reason:         row.Reason,
			ActorType:      account.ActorType(row.ActorType),
			ActorID:        row.ActorID,
			CreatedAt:      row.CreatedAt,
		}
	}

	return histories, nil
}

func (r *accountRepository) CreateDiscrepancyRecord(ctx context.Context, params account.CreateDiscrepancyRecordParams) (*account.DiscrepancyRecord, error) {
	row, err := r.queries.InsertDiscrepancyRecord(ctx, sqlc.InsertDiscrepancyRecordParams{
		AccountID:            params.AccountID,
		RawLedgerSum:         params.RawLedgerSum,
		SnapshotBalance:      params.SnapshotBalance,
		CalculatedTotal:      params.CalculatedTotal,
		DiscrepancyAmount:    params.DiscrepancyAmount,
		LastSnapshotSequence: params.LastSnapshotSequence,
		CurrentMaxSequence:   params.CurrentMaxSequence,
	})
	if err != nil {
		return nil, err
	}

	return mapToDiscrepancyDomain(&row), nil
}

func (r *accountRepository) ListUnresolvedDiscrepancies(ctx context.Context) ([]account.DiscrepancyRecord, error) {
	rows, err := r.queries.ListUnresolvedDiscrepancies(ctx)
	if err != nil {
		return nil, err
	}

	records := make([]account.DiscrepancyRecord, len(rows))
	for i := range rows {
		records[i] = *mapToDiscrepancyDomain(&rows[i])
	}

	return records, nil
}

func (r *accountRepository) ResolveDiscrepancy(ctx context.Context, id uuid.UUID, notes string) error {
	return r.queries.ResolveDiscrepancy(ctx, sqlc.ResolveDiscrepancyParams{
		ID:    id,
		Notes: pgtype.Text{String: notes, Valid: notes != ""},
	})
}

// CreateBalanceSnapshot checkpoints the account's balance at its latest entry.
// It returns account.ErrSnapshotExists when no entries were posted since the
// previous snapshot, because (account_id, last_entry_sequence) is unique.
func (r *accountRepository) CreateBalanceSnapshot(ctx context.Context, accountID uuid.UUID) (*account.BalanceSnapshot, error) {
	row, err := r.queries.CreateBalanceSnapshot(ctx, accountID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, account.ErrSnapshotExists
		}

		return nil, err
	}

	return mapToSnapshotDomain(&row), nil
}

func (r *accountRepository) GetLatestSnapshot(ctx context.Context, accountID uuid.UUID) (*account.BalanceSnapshot, error) {
	row, err := r.queries.GetLatestBalanceSnapshot(ctx, accountID)
	if err != nil {
		return nil, notFound(err, account.ErrSnapshotNotFound)
	}

	return mapToSnapshotDomain(&row), nil
}
