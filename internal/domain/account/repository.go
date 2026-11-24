package account

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines data access capabilities required by the Account domain.
// Implemented in internal/platform/database/postgres/repository/account_repo.go.
// Single-account lookups return ErrAccountNotFound when the account does not exist.
type Repository interface {
	Create(ctx context.Context, params CreateAccountParams) (*Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
	GetByOwnerTypeCurrency(ctx context.Context, params GetAccountByOwnerTypeCurrencyParams) (*Account, error)
	// List returns one page of accounts; Count returns how many match the
	// same filters, ignoring Page and Limit.
	List(ctx context.Context, params ListAccountsParams) ([]Account, error)
	Count(ctx context.Context, params ListAccountsParams) (int64, error)

	// Balance & Audit queries
	GetBalance(ctx context.Context, accountID uuid.UUID) (*GetAccountBalanceResult, error)
	GetStatementSummary(ctx context.Context, params GetStatementSummaryParams) (*StatementSummary, error)
	ListStatementEntries(ctx context.Context, params GetStatementEntriesParams) ([]StatementEntry, error)
	Reconcile(ctx context.Context, accountID uuid.UUID) (*ReconciliationResult, error)
	// ListAccountIDs returns up to limit account ids greater than after, in id order.
	ListAccountIDs(ctx context.Context, after uuid.NullUUID, limit int32) ([]uuid.UUID, error)

	// Status management
	ChangeAccountStatus(ctx context.Context, params StatusChangeParams) (*StatusHistory, error)
	GetAccountStatus(ctx context.Context, accountID uuid.UUID) (Status, error)
	ListAccountStatusHistory(ctx context.Context, params ListAccountStatusHistoryParams) ([]StatusHistory, error)

	// Discrepancy audit trail
	CreateDiscrepancyRecord(ctx context.Context, params CreateDiscrepancyRecordParams) (*DiscrepancyRecord, error)
	ListUnresolvedDiscrepancies(ctx context.Context, params ListDiscrepanciesParams) ([]DiscrepancyRecord, error)
	ResolveDiscrepancy(ctx context.Context, id uuid.UUID, notes string) error

	// Snapshots
	CreateBalanceSnapshot(ctx context.Context, accountID uuid.UUID) (*BalanceSnapshot, error)
	GetLatestSnapshot(ctx context.Context, accountID uuid.UUID) (*BalanceSnapshot, error)
}
