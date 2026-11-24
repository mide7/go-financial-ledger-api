package transaction

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists transactions. Implementations live in the platform layer.
type Repository interface {
	// Create posts a transaction. It returns ErrAccountNotPostable when an
	// entry's account status forbids that entry.
	Create(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	// GetByID returns the transaction with its entries, or ErrTransactionNotFound.
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	List(ctx context.Context, params ListTransactionsParams) ([]Transaction, error)
	// Reverse posts the mirror of transaction params.ID as a TypeReversal
	// transaction. It returns ErrAlreadyReversed or ErrCannotReverseReversal.
	Reverse(ctx context.Context, params ReverseTransactionParams) (*Transaction, error)
}
