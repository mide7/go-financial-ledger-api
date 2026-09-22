package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
)

type CreateAccountDTO struct {
	OwnerID  string `json:"owner_id" validate:"required,min=1,max=64"`
	Type     string `json:"type" validate:"required,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUE MERCHANT_PAYABLE"`
	Currency string `json:"currency" validate:"required,len=3,uppercase"`
}

func (c CreateAccountDTO) ToDomainParams() account.CreateAccountParams {
	return account.CreateAccountParams{
		OwnerID:  c.OwnerID,
		Type:     c.Type,
		Currency: c.Currency,
	}
}

type GetAccountDetailsDTO struct {
	ID uuid.UUID `form:"id" validate:"required,uuid"`
}

type ListAccountsDTO struct {
	Page     int    `form:"page" validate:"omitempty,gte=1"`
	Limit    int    `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Currency string `form:"currency" validate:"omitempty,len=3,uppercase"`
	Type     string `form:"type" validate:"omitempty,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUE MERCHANT_PAYABLE"`
}

func (l ListAccountsDTO) ToDomainParams() account.ListAccountsParams {
	return account.ListAccountsParams{
		Page:     l.Page,
		Limit:    l.Limit,
		Currency: l.Currency,
		Type:     l.Type,
	}
}

type GetAccountBalanceDTO struct {
	ID uuid.UUID `form:"id" validate:"required,uuid"`
}

type GetAccountStatementSummaryDTO struct {
	ID        uuid.UUID `form:"id" validate:"required,uuid"`
	StartDate time.Time `form:"start_date" validate:"required"`
	EndDate   time.Time `form:"end_date" validate:"required"`
}

func (g GetAccountStatementSummaryDTO) ToDomainParams() account.GetStatementSummaryParams {
	return account.GetStatementSummaryParams{
		AccountID: g.ID,
		StartDate: g.StartDate,
		EndDate:   g.EndDate,
	}
}

type GetAccountStatementEntriesDTO struct {
	ID        uuid.UUID     `form:"id" validate:"required,uuid"`
	StartDate time.Time     `form:"start_date" validate:"required"`
	EndDate   time.Time     `form:"end_date" validate:"required"`
	Limit     int32         `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Cursor    uuid.NullUUID `form:"cursor" validate:"omitempty,uuid"`
}

func (g GetAccountStatementEntriesDTO) ToDomainParams() account.GetStatementEntriesParams {
	return account.GetStatementEntriesParams{
		AccountID: g.ID,
		StartDate: g.StartDate,
		EndDate:   g.EndDate,
		Limit:     g.Limit,
		Cursor:    g.Cursor,
	}
}

type ReconcileAccountDTO struct {
	ID            uuid.UUID `form:"id" validate:"required,uuid"`
	AsOfTimestamp time.Time `json:"as_of_timestamp" validate:"omitempty"`
}

func (r ReconcileAccountDTO) ToDomainParams() account.ReconcileAccountParams {
	return account.ReconcileAccountParams{
		ID:            r.ID,
		AsOfTimestamp: r.AsOfTimestamp,
	}
}

type CreateAccountSnapshotDTO struct {
	ID                  uuid.UUID `form:"id" validate:"required,uuid"`
	TargetEntrySequence int64     `json:"target_entry_sequence" validate:"omitempty,gte=1"`
	Reason              string    `json:"reason" validate:"required,min=1,max=128"`
}

func (c CreateAccountSnapshotDTO) ToDomainParams() account.CreateAccountSnapshotParams {
	return account.CreateAccountSnapshotParams{
		ID:                  c.ID,
		TargetEntrySequence: c.TargetEntrySequence,
		Reason:              c.Reason,
	}
}
