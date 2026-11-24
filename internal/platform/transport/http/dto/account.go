// Package dto defines HTTP request bodies and parameters, with validation tags.
//
// Fields typed uuid.UUID or time.Time are parsed by the request decoder, so a
// malformed value fails decoding before validation runs; their tags only need
// to check presence.
package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
)

// CreateAccountDTO is the body of POST /accounts.
type CreateAccountDTO struct {
	OwnerID  string `json:"owner_id" validate:"required,min=1,max=64"`
	Type     string `json:"type" validate:"required,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUE MERCHANT_PAYABLE"`
	Currency string `json:"currency" validate:"required,len=3,uppercase"`
}

// ToDomainParams converts the request to service parameters.
func (c CreateAccountDTO) ToDomainParams() account.CreateAccountParams {
	return account.CreateAccountParams{
		OwnerID:  c.OwnerID,
		Type:     c.Type,
		Currency: c.Currency,
	}
}

// GetAccountDetailsDTO holds the path parameters of GET /accounts/{id}.
type GetAccountDetailsDTO struct {
	ID uuid.UUID `form:"id" validate:"required"`
}

// ListAccountsDTO holds the query parameters of GET /accounts. Page is 1-based;
// omitted page and limit fall back to the service defaults.
type ListAccountsDTO struct {
	Page     int    `form:"page" validate:"omitempty,gte=1"`
	Limit    int    `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Currency string `form:"currency" validate:"omitempty,len=3,uppercase"`
	Type     string `form:"type" validate:"omitempty,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUE MERCHANT_PAYABLE"`
}

// ToDomainParams converts the request to service parameters.
func (l ListAccountsDTO) ToDomainParams() account.ListAccountsParams {
	return account.ListAccountsParams{
		Page:     l.Page,
		Limit:    l.Limit,
		Currency: l.Currency,
		Type:     l.Type,
	}
}

// GetAccountBalanceDTO holds the path parameters of GET /accounts/{id}/balance.
type GetAccountBalanceDTO struct {
	ID uuid.UUID `form:"id" validate:"required"`
}

// GetAccountStatementSummaryDTO holds the path and query parameters of
// GET /accounts/{id}/statement/summary. Dates are RFC 3339.
type GetAccountStatementSummaryDTO struct {
	ID        uuid.UUID `form:"id" validate:"required"`
	StartDate time.Time `form:"start_date" validate:"required"`
	EndDate   time.Time `form:"end_date" validate:"required"`
}

// ToDomainParams converts the request to service parameters.
func (g GetAccountStatementSummaryDTO) ToDomainParams() account.GetStatementSummaryParams {
	return account.GetStatementSummaryParams{
		AccountID: g.ID,
		StartDate: g.StartDate,
		EndDate:   g.EndDate,
	}
}

// GetAccountStatementEntriesDTO holds the path and query parameters of
// GET /accounts/{id}/statement/entries. Dates are RFC 3339; cursor is the ID
// of the last entry on the previous page.
type GetAccountStatementEntriesDTO struct {
	ID        uuid.UUID     `form:"id" validate:"required"`
	StartDate time.Time     `form:"start_date" validate:"required"`
	EndDate   time.Time     `form:"end_date" validate:"required"`
	Limit     int32         `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Cursor    uuid.NullUUID `form:"cursor"`
}

// ToDomainParams converts the request to service parameters.
func (g GetAccountStatementEntriesDTO) ToDomainParams() account.GetStatementEntriesParams {
	return account.GetStatementEntriesParams{
		AccountID: g.ID,
		StartDate: g.StartDate,
		EndDate:   g.EndDate,
		Limit:     g.Limit,
		Cursor:    g.Cursor,
	}
}

// ReconcileAccountDTO holds the parameters of POST /accounts/{id}/reconcile.
// Reconciliation always runs against the current ledger; as_of_timestamp is
// rejected until point-in-time reconciliation is supported.
type ReconcileAccountDTO struct {
	ID            uuid.UUID `form:"id" validate:"required"`
	AsOfTimestamp time.Time `json:"as_of_timestamp" validate:"isdefault"`
}

// CreateAccountSnapshotDTO holds the parameters of POST /accounts/{id}/snapshot.
// A snapshot always covers every entry posted so far; target_entry_sequence is
// rejected until snapshots at an earlier sequence are supported.
type CreateAccountSnapshotDTO struct {
	ID                  uuid.UUID `form:"id" validate:"required"`
	TargetEntrySequence int64     `json:"target_entry_sequence" validate:"isdefault"`
	Reason              string    `json:"reason" validate:"required,min=1,max=128"`
}

// ToDomainParams converts the request to service parameters.
func (c CreateAccountSnapshotDTO) ToDomainParams() account.CreateAccountSnapshotParams {
	return account.CreateAccountSnapshotParams{
		ID:     c.ID,
		Reason: c.Reason,
	}
}
