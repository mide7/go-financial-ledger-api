package services

import (
	"context"

	"github.com/mide7/go-financial-ledger-api/internal/domain/model"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/db"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
)

type ITransactionService interface {
	CreateTransaction(ctx context.Context, dto dto.CreateTransactionDTO) (*model.Transaction, error)
	GetTransactionDetails(ctx context.Context, id string) (*model.Transaction, error)
	ListTransactions(ctx context.Context, dto dto.ListTransactionsDTO) ([]model.Transaction, error)
	ReverseTransaction(ctx context.Context, id string) error
	GetTransactionEntries(ctx context.Context, id string) ([]model.Entry, error)
}

type TransactionService struct {
	db db.Querier
}

func NewTransactionService(db db.Querier) *TransactionService {
	return &TransactionService{
		db: db,
	}
}

func (s *TransactionService) CreateTransaction(ctx context.Context, dto dto.CreateTransactionDTO) (*model.Transaction, error) {
	return nil, nil
}

func (s *TransactionService) GetTransactionDetails(ctx context.Context, id string) (*model.Transaction, error) {
	return nil, nil
}

func (s *TransactionService) ListTransactions(ctx context.Context, dto dto.ListTransactionsDTO) ([]model.Transaction, error) {
	return nil, nil
}

func (s *TransactionService) ReverseTransaction(ctx context.Context, id string) error {
	return nil
}

func (s *TransactionService) GetTransactionEntries(ctx context.Context, id string) ([]model.Entry, error) {
	return nil, nil
}
