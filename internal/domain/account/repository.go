package account

import (
	"context"
)

// Repository defines data access capabilities required by the Account domain.
// Implemented in internal/platform/database/postgres/repository/account.go
type Repository interface {
	Create(ctx context.Context, params CreateAccountParams) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByOwnerTypeCurrency(ctx context.Context, ownerID, accountType, currency string) (*Account, error)
	List(ctx context.Context, params ListAccountsParams) ([]Account, error)

	// Balance & Audit queries
	GetBalance(ctx context.Context, accountID string) (int64, error)
	GetStatement(ctx context.Context, accountID string) ([]StatementEntry, error)
	Reconcile(ctx context.Context, accountID string) (*ReconciliationResult, error)
}
