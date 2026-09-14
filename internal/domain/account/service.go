package account

import (
	"context"
)

type Service interface {
	CreateAccount(ctx context.Context, params CreateAccountParams) (*Account, error)
	GetAccountDetails(ctx context.Context, id string) (*Account, error)
	ListAccounts(ctx context.Context, params ListAccountsParams) ([]Account, error)
	GetAccountBalance(ctx context.Context, id string) (int64, error)
	GetAccountStatement(ctx context.Context, id string) ([]Statement, error)
	GetAccountEntries(ctx context.Context, id string) ([]StatementEntry, error)
	ReconcileAccount(ctx context.Context, id string) (*ReconciliationResult, error)
	CreateAccountSnapshot(ctx context.Context, params CreateAccountSnapshotParams) (*Account, error)
}

type accountService struct {
	repo Repository
}

func NewAccountService(repo Repository) Service {
	return &accountService{
		repo: repo,
	}
}

func (s *accountService) CreateAccount(ctx context.Context, params CreateAccountParams) (*Account, error) {
	return nil, nil
}

func (s *accountService) GetAccountDetails(ctx context.Context, id string) (*Account, error) {
	return nil, nil
}

func (s *accountService) ListAccounts(ctx context.Context, params ListAccountsParams) ([]Account, error) {
	return nil, nil
}

func (s *accountService) GetAccountBalance(ctx context.Context, id string) (int64, error) {
	return 0, nil
}

func (s *accountService) GetAccountStatement(ctx context.Context, id string) ([]Statement, error) {
	return nil, nil
}

func (s *accountService) GetAccountEntries(ctx context.Context, id string) ([]StatementEntry, error) {
	return nil, nil
}

func (s *accountService) ReconcileAccount(ctx context.Context, id string) (*ReconciliationResult, error) {
	return nil, nil
}

func (s *accountService) CreateAccountSnapshot(ctx context.Context, params CreateAccountSnapshotParams) (*Account, error) {
	return nil, nil
}
