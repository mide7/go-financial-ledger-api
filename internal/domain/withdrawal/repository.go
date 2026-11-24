package withdrawal

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists withdrawals. There is no implementation yet; see docs/todo.md.
type Repository interface {
	// Create posts the lock transaction and inserts the withdrawal in one database transaction.
	Create(ctx context.Context, params CreateWithdrawalParams) (*Withdrawal, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Withdrawal, error)
	List(ctx context.Context, params ListWithdrawalsParams) ([]Withdrawal, error)
	// Settle and Fail post the resolution transaction and return ErrNotProcessing
	// if the withdrawal was already resolved.
	Settle(ctx context.Context, params SettleWithdrawalParams) (*Withdrawal, error)
	Fail(ctx context.Context, params FailWithdrawalParams) (*Withdrawal, error)
}
