package transaction

import (
	"context"
)

type Service interface {
	CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	GetTransactionDetails(ctx context.Context, id string) (*Transaction, error)
	ListTransactions(ctx context.Context, params ListTransactionsParams) ([]Transaction, error)
	ReverseTransaction(ctx context.Context, id string) error
}

type transactionService struct {
	repo Repository
}

func NewTransactionService(repo Repository) Service {
	return &transactionService{
		repo: repo,
	}
}

func (s *transactionService) CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, error) {
	return nil, nil
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
