package transaction

import (
	"context"

	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// Service is the transaction use-case API consumed by the transport layer.
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

// NewTransactionService returns a Service backed by repo and currencyRegistry.
func NewTransactionService(repo Repository, currencyRegistry currency.Registry) Service {
	return &transactionService{
		repo:             repo,
		currencyRegistry: currencyRegistry,
	}
}

func (s *transactionService) CreateTransaction(
	ctx context.Context,
	params CreateTransactionParams,
) (*Transaction, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	if _, active := s.currencyRegistry.Get(params.Currency); !active {
		return nil, ErrInvalidCurrency
	}

	return s.repo.Create(ctx, params)
}

func (s *transactionService) GetTransactionDetails(_ context.Context, _ string) (*Transaction, error) {
	return nil, domain.ErrNotImplemented
}

func (s *transactionService) ListTransactions(_ context.Context, _ ListTransactionsParams) ([]Transaction, error) {
	return nil, domain.ErrNotImplemented
}

func (s *transactionService) ReverseTransaction(_ context.Context, _ string) error {
	return domain.ErrNotImplemented
}
