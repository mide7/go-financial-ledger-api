package withdrawal

import (
	"context"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
)

// Service is the withdrawal use-case API consumed by the transport layer.
type Service interface {
	CreateWithdrawal(ctx context.Context, params CreateWithdrawalParams) (*Withdrawal, error)
	GetWithdrawal(ctx context.Context, id uuid.UUID) (*Withdrawal, error)
	ListWithdrawals(ctx context.Context, params ListWithdrawalsParams) ([]Withdrawal, error)
	SettleWithdrawal(ctx context.Context, params SettleWithdrawalParams) (*Withdrawal, error)
	FailWithdrawal(ctx context.Context, params FailWithdrawalParams) (*Withdrawal, error)
}

// TODO(later-withdrawals): add a Repository field once the withdrawals table exists.
type withdrawalService struct{}

// NewWithdrawalService returns a Service whose methods all return
// domain.ErrNotImplemented until withdrawals are built.
func NewWithdrawalService() Service {
	return &withdrawalService{}
}

func (s *withdrawalService) CreateWithdrawal(_ context.Context, _ CreateWithdrawalParams) (*Withdrawal, error) {
	return nil, domain.ErrNotImplemented
}

func (s *withdrawalService) GetWithdrawal(_ context.Context, _ uuid.UUID) (*Withdrawal, error) {
	return nil, domain.ErrNotImplemented
}

func (s *withdrawalService) ListWithdrawals(_ context.Context, _ ListWithdrawalsParams) ([]Withdrawal, error) {
	return nil, domain.ErrNotImplemented
}

func (s *withdrawalService) SettleWithdrawal(_ context.Context, _ SettleWithdrawalParams) (*Withdrawal, error) {
	return nil, domain.ErrNotImplemented
}

func (s *withdrawalService) FailWithdrawal(_ context.Context, _ FailWithdrawalParams) (*Withdrawal, error) {
	return nil, domain.ErrNotImplemented
}
