package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

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

func (r *accountRepository) GetStatementEntries(ctx context.Context, params account.GetStatementEntriesParams) ([]account.StatementEntry, error) {
	return nil, nil
}

func (r *accountRepository) Reconcile(ctx context.Context, accountID string) (*account.ReconciliationResult, error) {
	return nil, nil
}

// Freeze / hold management
func (r *accountRepository) FreezeAccount(ctx context.Context, accountID, reason string) error {
	return nil
}
func (r *accountRepository) UnfreezeAccount(ctx context.Context, accountID, resolutionNotes string) error {
	return nil
}
func (r *accountRepository) IsFrozen(ctx context.Context, accountID string) (bool, error) {
	return false, nil
}
func (r *accountRepository) GetFrozenAccounts(ctx context.Context, accountIDs []string) ([]string, error) {
	return nil, nil
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
