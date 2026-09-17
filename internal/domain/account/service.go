package account

import (
	"context"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

type Service interface {
	CreateAccount(ctx context.Context, params CreateAccountParams) (*Account, error)
	GetAccountDetails(ctx context.Context, id uuid.UUID) (*Account, error)
	ListAccounts(ctx context.Context, params ListAccountsParams) ([]Account, error)
	GetAccountBalance(ctx context.Context, id uuid.UUID) (int64, error)
	GetAccountStatement(ctx context.Context, id uuid.UUID) ([]Statement, error)
	GetAccountEntries(ctx context.Context, id uuid.UUID) ([]StatementEntry, error)
	ReconcileAccount(ctx context.Context, id uuid.UUID) (*ReconciliationResult, error)
	CreateAccountSnapshot(ctx context.Context, params CreateAccountSnapshotParams) (*Account, error)
}

type accountService struct {
	repo             Repository
	currencyRegistry currency.Registry
}

func NewAccountService(repo Repository, currencyRegistry currency.Registry) Service {
	return &accountService{
		repo:             repo,
		currencyRegistry: currencyRegistry,
	}
}

func (s *accountService) CreateAccount(ctx context.Context, params CreateAccountParams) (*Account, error) {
	return nil, nil
}

func (s *accountService) GetAccountDetails(ctx context.Context, id uuid.UUID) (*Account, error) {
	return nil, nil
}

func (s *accountService) ListAccounts(ctx context.Context, params ListAccountsParams) ([]Account, error) {
	return nil, nil
}

func (s *accountService) GetAccountBalance(ctx context.Context, id uuid.UUID) (int64, error) {
	return 0, nil
}

func (s *accountService) GetAccountStatement(ctx context.Context, id uuid.UUID) ([]Statement, error) {
	return nil, nil
}

func (s *accountService) GetAccountEntries(ctx context.Context, id uuid.UUID) ([]StatementEntry, error) {
	return nil, nil
}

func (s *accountService) ReconcileAccount(ctx context.Context, id uuid.UUID) (*ReconciliationResult, error) {
	return nil, nil
}

func (s *accountService) CreateAccountSnapshot(ctx context.Context, params CreateAccountSnapshotParams) (*Account, error) {
	return nil, nil
}
