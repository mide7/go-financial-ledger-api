package services

import (
	"context"

	"github.com/mide7/go-financial-ledger-api/internal/domain/model"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/db"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
)

type IAccountService interface {
	CreateAccount(ctx context.Context, dto dto.CreateAccountDTO) (*model.Account, error)
	GetAccountDetails(ctx context.Context, id string) (*model.Account, error)
	ListAccounts(ctx context.Context, dto dto.ListAccountsDTO) ([]model.Account, error)
	GetAccountBalance(ctx context.Context, id string) (float64, error)
	GetAccountStatement(ctx context.Context, id string) ([]model.Transaction, error)
	GetAccountEntries(ctx context.Context, id string) ([]model.Entry, error)
}

type AccountService struct {
	db db.Querier
}

func NewAccountService(db db.Querier) *AccountService {
	return &AccountService{
		db: db,
	}
}

func (s *AccountService) CreateAccount(ctx context.Context, dto dto.CreateAccountDTO) (*model.Account, error) {
	return nil, nil
}

func (s *AccountService) GetAccountDetails(ctx context.Context, id string) (*model.Account, error) {
	return nil, nil
}

func (s *AccountService) ListAccounts(ctx context.Context, dto dto.ListAccountsDTO) ([]model.Account, error) {
	return nil, nil
}

func (s *AccountService) GetAccountBalance(ctx context.Context, id string) (float64, error) {
	return 0, nil
}

func (s *AccountService) GetAccountStatement(ctx context.Context, id string) ([]model.Transaction, error) {
	return nil, nil
}

func (s *AccountService) GetAccountEntries(ctx context.Context, id string) ([]model.Entry, error) {
	return nil, nil
}
