package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
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
func (r *accountRepository) GetBalance(ctx context.Context, accountID string) (*account.GetAccountBalanceResult, error) {
	return nil, nil
}

func (r *accountRepository) GetStatementSummary(ctx context.Context, params account.GetStatementSummaryParams) (*account.StatementSummary, error) {
	return nil, nil
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
