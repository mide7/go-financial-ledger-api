package transaction

import "context"

// Repository persists transactions. Implementations live in the platform layer.
type Repository interface {
	Create(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	GetByID(ctx context.Context, id string) (*Transaction, error)
	List(ctx context.Context, params ListTransactionsParams) ([]Transaction, error)
	Reverse(ctx context.Context, id string) error
}
