package transaction

import (
	"context"

	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

type Service interface {
	CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	GetTransactionDetails(ctx context.Context, id string) (*Transaction, error)
	ListTransactions(ctx context.Context, params ListTransactionsParams) ([]Transaction, error)
	ReverseTransaction(ctx context.Context, id string) error
}

type transactionService struct {
	repo             Repository
	currencyRegistry currency.Registry
}

func NewTransactionService(repo Repository, currencyRegistry currency.Registry) Service {
	return &transactionService{
		repo:             repo,
		currencyRegistry: currencyRegistry,
	}
}

func (s *transactionService) CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, error) {
	// 1. Execute DTO and domain invariant checks (entry counts, zero amounts, double-entry balancing)
	if err := params.Validate(); err != nil {
		return nil, err
	}

	// 2. Validate currency against the in-memory registry (0ms latency lookup)
	_, active := s.currencyRegistry.Get(params.Currency)
	if !active {
		return nil, ErrInvalidCurrency
	}

	// 3. Delegate atomic persistence (header + batch entries) to the repository layer
	createdTx, err := s.repo.Create(ctx, params)
	if err != nil {
		return nil, err
	}

	return createdTx, nil
}

func (s *transactionService) GetTransactionDetails(ctx context.Context, id string) (*Transaction, error) {
	return nil, nil
}

func (s *transactionService) ListTransactions(ctx context.Context, params ListTransactionsParams) ([]Transaction, error) {
	return nil, nil
}

func (s *transactionService) ReverseTransaction(ctx context.Context, id string) error {
	return nil
}
