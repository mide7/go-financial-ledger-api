package dto

import (
	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/withdrawal"
)

// CreateWithdrawalDTO is the body of POST /withdrawals. reference is the unique
// reference of the transaction that locks the funds.
type CreateWithdrawalDTO struct {
	AccountID uuid.UUID `json:"account_id" validate:"required"`
	Amount    int64     `json:"amount" validate:"required,gt=0"`
	Processor string    `json:"processor" validate:"required,min=1,max=64"`
	Reference string    `json:"reference" validate:"required,min=1,max=128"`
}

// ToDomainParams converts the request to service parameters.
func (c CreateWithdrawalDTO) ToDomainParams() withdrawal.CreateWithdrawalParams {
	return withdrawal.CreateWithdrawalParams{
		AccountID: c.AccountID,
		Amount:    c.Amount,
		Processor: c.Processor,
		Reference: c.Reference,
	}
}

// GetWithdrawalDTO holds the path parameters of GET /withdrawals/{id}.
type GetWithdrawalDTO struct {
	ID uuid.UUID `form:"id" validate:"required"`
}

// ListWithdrawalsDTO holds the query parameters of GET /withdrawals, newest first.
type ListWithdrawalsDTO struct {
	AccountID uuid.NullUUID `form:"account_id"`
	Status    string        `form:"status" validate:"omitempty,oneof=PROCESSING SETTLED FAILED"`
	Cursor    uuid.NullUUID `form:"cursor"`
	Limit     int32         `form:"limit" validate:"omitempty,gte=1,lte=100"`
}

// ToDomainParams converts the request to service parameters.
func (l ListWithdrawalsDTO) ToDomainParams() withdrawal.ListWithdrawalsParams {
	return withdrawal.ListWithdrawalsParams{
		AccountID: l.AccountID,
		Status:    withdrawal.Status(l.Status),
		Cursor:    l.Cursor,
		Limit:     l.Limit,
	}
}

// SettleWithdrawalDTO holds the parameters of POST /withdrawals/{id}/settle.
type SettleWithdrawalDTO struct {
	ID                 uuid.UUID `form:"id" validate:"required"`
	ProcessorReference string    `json:"processor_reference" validate:"required,min=1,max=128"`
}

// ToDomainParams converts the request to service parameters.
func (s SettleWithdrawalDTO) ToDomainParams() withdrawal.SettleWithdrawalParams {
	return withdrawal.SettleWithdrawalParams{ID: s.ID, ProcessorReference: s.ProcessorReference}
}

// FailWithdrawalDTO holds the parameters of POST /withdrawals/{id}/fail.
type FailWithdrawalDTO struct {
	ID     uuid.UUID `form:"id" validate:"required"`
	Reason string    `json:"reason" validate:"required,min=1,max=256"`
}

// ToDomainParams converts the request to service parameters.
func (f FailWithdrawalDTO) ToDomainParams() withdrawal.FailWithdrawalParams {
	return withdrawal.FailWithdrawalParams{ID: f.ID, Reason: f.Reason}
}
