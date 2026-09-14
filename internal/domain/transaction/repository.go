package transaction

import "context"

type Repository interface {
	Create(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	GetByID(ctx context.Context, id string) (*Transaction, error)
	List(ctx context.Context, params ListTransactionsParams) ([]Transaction, error)
	Reverse(ctx context.Context, id string) error
}
