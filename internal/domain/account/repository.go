package account

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines data access capabilities required by the Account domain.
// Implemented in internal/platform/database/postgres/repository/account.go
type Repository interface {
	Create(ctx context.Context, params CreateAccountParams) (*Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
	GetByOwnerTypeCurrency(ctx context.Context, params GetAccountByOwnerTypeCurrencyParams) (*Account, error)
	List(ctx context.Context, params ListAccountsParams) ([]Account, error)

	// Balance & Audit queries
	GetBalance(ctx context.Context, accountID uuid.UUID) (*GetAccountBalanceResult, error)
	GetStatementSummary(ctx context.Context, params GetStatementSummaryParams) (*StatementSummary, error)
	GetStatementEntries(ctx context.Context, params GetStatementEntriesParams) ([]StatementEntry, error)
	Reconcile(ctx context.Context, accountID string) (*ReconciliationResult, error)

	// Freeze / hold management
	FreezeAccount(ctx context.Context, accountID, reason string) error
	UnfreezeAccount(ctx context.Context, accountID, resolutionNotes string) error
	IsFrozen(ctx context.Context, accountID string) (bool, error)
	GetFrozenAccounts(ctx context.Context, accountIDs []string) ([]string, error)

	// Discrepancy audit trail
	InsertDiscrepancyRecord(ctx context.Context, r *ReconciliationResult) (*DiscrepancyRecord, error)
	ListUnresolvedDiscrepancies(ctx context.Context) ([]DiscrepancyRecord, error)
	ResolveDiscrepancy(ctx context.Context, id, notes string) error

	// Snapshots
	CreateBalanceSnapshot(ctx context.Context, accountID string) (*BalanceSnapshot, error)
	GetLatestSnapshot(ctx context.Context, accountID string) (*BalanceSnapshot, error)
}
