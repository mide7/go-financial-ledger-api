package repository

import (
	"context"
	"database/sql"
	"errors"
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
	account, err := r.queries.GetAccountByID(ctx, id)

	if err != nil {
		return nil, err
	}

	return mapToAccountDomain(&account), nil
}

func (r *accountRepository) GetByOwnerTypeCurrency(ctx context.Context, params account.GetAccountByOwnerTypeCurrencyParams) (*account.Account, error) {
	account, err := r.queries.GetAccountByOwnerTypeCurrency(ctx, sqlc.GetAccountByOwnerTypeCurrencyParams{
		OwnerID:  params.OwnerID,
		Type:     params.Type,
		Currency: params.Currency,
	})

	if err != nil {
		return nil, err
	}

	return mapToAccountDomain(&account), nil
}

func (r *accountRepository) List(ctx context.Context, params account.ListAccountsParams) ([]account.Account, error) {
	accounts, err := r.queries.ListAccounts(ctx, sqlc.ListAccountsParams{
		OwnerID:   pgtype.Text{String: params.OwnerID, Valid: params.OwnerID != ""},
		Type:      pgtype.Text{String: params.Type, Valid: params.Type != ""},
		Currency:  pgtype.Text{String: params.Currency, Valid: params.Currency != ""},
		SortBy:    pgtype.Text{String: params.SortBy, Valid: params.SortBy != ""},
		SortOrder: pgtype.Text{String: params.OrderBy, Valid: params.OrderBy != ""},
		Offset:    int32(params.Page * params.Limit),
		Limit:     int32(params.Limit),
	})

	if err != nil {
		return nil, err
	}

	return mapToAccountDomainList(accounts), nil
}

// Balance & Audit queries
func (r *accountRepository) GetBalance(ctx context.Context, accountID uuid.UUID) (*account.GetAccountBalanceResult, error) {
	balance, err := r.queries.GetAccountBalance(ctx, accountID)

	if err != nil {
		return nil, err
	}

	return &account.GetAccountBalanceResult{
		Currency:     balance.Currency,
		Exponent:     balance.Exponent,
		BalanceMinor: balance.BalanceMinor,
		BalanceMajor: balance.BalanceMajor,
	}, nil
}

func (r *accountRepository) GetStatementSummary(ctx context.Context, params account.GetStatementSummaryParams) (*account.StatementSummary, error) {
	summary, err := r.queries.GetAccountStatementSummary(ctx, sqlc.GetAccountStatementSummaryParams{
		AccountID: params.AccountID,
		StartDate: pgtype.Timestamptz{Time: params.StartDate, Valid: true},
		EndDate:   pgtype.Timestamptz{Time: params.EndDate, Valid: true},
	})

	if err != nil {
		return nil, err
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
		Limit:     int32(params.Limit),
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
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	lockedAccount, err := qtx.GetAccountForUpdate(ctx, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, account.ErrAccountNotFound
	} else if err != nil {
		return nil, err
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

// Status management
func (r *accountRepository) ChangeAccountStatus(ctx context.Context, params account.StatusChangeParams) (*account.StatusHistory, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	lockedAccount, err := qtx.GetAccountForUpdate(ctx, params.AccountID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, account.ErrAccountNotFound
	} else if err != nil {
		return nil, err
	}

	if params.ExpectedStatus != nil && *params.ExpectedStatus != account.AccountStatus(lockedAccount.Status) {
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
		NewStatus:      account.AccountStatus(result.NewStatus),
		Reason:         result.Reason,
		ActorType:      account.ActorType(result.ActorType),
		ActorID:        result.ActorID,
		CreatedAt:      result.CreatedAt,
	}, nil
}

func (r *accountRepository) GetAccountStatus(ctx context.Context, accountID uuid.UUID) (account.AccountStatus, error) {
	status, err := r.queries.GetAccountStatus(ctx, accountID)
	if err != nil {
		return "", err
	}

	return account.AccountStatus(status), nil
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
			NewStatus:      account.AccountStatus(row.NewStatus),
			Reason:         row.Reason,
			ActorType:      account.ActorType(row.ActorType),
			ActorID:        row.ActorID,
			CreatedAt:      row.CreatedAt,
		}
	}

	return histories, nil
}

// Discrepancy audit trail
func (r *accountRepository) InsertDiscrepancyRecord(ctx context.Context, params *account.ReconciliationResult) (*account.DiscrepancyRecord, error) {
	return nil, nil
}
func (r *accountRepository) ListUnresolvedDiscrepancies(ctx context.Context) ([]account.DiscrepancyRecord, error) {
	return nil, nil
}
func (r *accountRepository) ResolveDiscrepancy(ctx context.Context, id, notes string) error {
	return nil
}

// Snapshots
func (r *accountRepository) CreateBalanceSnapshot(ctx context.Context, accountID string) (*account.BalanceSnapshot, error) {
	return nil, nil
}
func (r *accountRepository) GetLatestSnapshot(ctx context.Context, accountID string) (*account.BalanceSnapshot, error) {
	return nil, nil
}
