package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
)

// EntryDTO is one leg of a transaction in a create request.
type EntryDTO struct {
	AccountID string `json:"account_id" validate:"required,uuid"`
	Type      string `json:"type" validate:"required,oneof=DEBIT CREDIT"`
	Amount    int64  `json:"amount" validate:"required,gt=0"`
}

// CreateTransactionDTO is the body of POST /transactions.
type CreateTransactionDTO struct {
	Reference           string          `json:"reference" validate:"required,min=1,max=128"`
	Type                string          `json:"type" validate:"required,min=1,max=64"`
	Currency            string          `json:"currency" validate:"required,len=3"`
	ParentTransactionID *string         `json:"parent_transaction_id" validate:"omitempty,uuid"`
	Description         string          `json:"description" validate:"required,max=256"`
	Metadata            json.RawMessage `json:"metadata"` // optional JSON object; defaults to {}
	TransactionDate     time.Time       `json:"transaction_date" validate:"required"`
	Entries             []EntryDTO      `json:"entries" validate:"required,min=2,dive"`
}

// ToDomainParams converts the request to service parameters. It fails if an ID is not a valid UUID.
func (d CreateTransactionDTO) ToDomainParams() (transaction.CreateTransactionParams, error) {
	entries := make([]transaction.EntryParams, len(d.Entries))
	for i, e := range d.Entries {
		accID, err := uuid.Parse(e.AccountID)
		if err != nil {
			return transaction.CreateTransactionParams{}, err
		}

		entries[i] = transaction.EntryParams{
			AccountID: accID,
			Type:      domain.EntryType(e.Type),
			Amount:    e.Amount,
		}
	}

	var parentID uuid.NullUUID

	if d.ParentTransactionID != nil && *d.ParentTransactionID != "" {
		parsed, err := uuid.Parse(*d.ParentTransactionID)
		if err != nil {
			return transaction.CreateTransactionParams{}, err
		}

		parentID = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	return transaction.CreateTransactionParams{
		Reference:           d.Reference,
		Type:                d.Type,
		Currency:            currency.Code(d.Currency),
		ParentTransactionID: parentID,
		Description:         d.Description,
		TransactionDate:     d.TransactionDate,
		Metadata:            d.Metadata,
		Entries:             entries,
	}, nil
}

// GetTransactionDetailsDTO holds the path parameters of GET /transactions/{id}.
type GetTransactionDetailsDTO struct {
	ID uuid.UUID `form:"id" validate:"required"`
}

// ListTransactionsDTO holds the query parameters of GET /transactions, newest
// first. Every filter is optional; start_date is inclusive and end_date
// exclusive. cursor is the id of the last transaction on the previous page.
type ListTransactionsDTO struct {
	AccountID uuid.NullUUID `form:"account_id"`
	Reference string        `form:"reference" validate:"omitempty,max=128"`
	Type      string        `form:"type" validate:"omitempty,max=64"`
	Currency  string        `form:"currency" validate:"omitempty,len=3,uppercase"`
	StartDate time.Time     `form:"start_date"`
	EndDate   time.Time     `form:"end_date"`
	Cursor    uuid.NullUUID `form:"cursor"`
	Limit     int32         `form:"limit" validate:"omitempty,gte=1,lte=100"`
}

// ToDomainParams converts the request to service parameters.
func (l ListTransactionsDTO) ToDomainParams() transaction.ListTransactionsParams {
	return transaction.ListTransactionsParams{
		AccountID: l.AccountID,
		Reference: l.Reference,
		Type:      l.Type,
		Currency:  currency.Code(l.Currency),
		StartDate: l.StartDate,
		EndDate:   l.EndDate,
		Cursor:    l.Cursor,
		Limit:     l.Limit,
	}
}

// ReverseTransactionDTO holds the parameters of POST /transactions/{id}/reverse.
// reference is the reversal's own unique reference.
type ReverseTransactionDTO struct {
	ID        uuid.UUID `form:"id" validate:"required"`
	Reference string    `json:"reference" validate:"required,min=1,max=128"`
	Reason    string    `json:"reason" validate:"required,min=1,max=256"`
}

// ToDomainParams converts the request to service parameters.
func (r ReverseTransactionDTO) ToDomainParams() transaction.ReverseTransactionParams {
	return transaction.ReverseTransactionParams{ID: r.ID, Reference: r.Reference, Reason: r.Reason}
}

// CreateConversionDTO is the body of POST /conversions. rate is target units
// per source unit, as a decimal string.
type CreateConversionDTO struct {
	Reference       string    `json:"reference" validate:"required,min=1,max=128"`
	SourceAccountID uuid.UUID `json:"source_account_id" validate:"required"`
	TargetAccountID uuid.UUID `json:"target_account_id" validate:"required"`
	SourceAmount    int64     `json:"source_amount" validate:"required,gt=0"`
	Rate            string    `json:"rate" validate:"required,numeric"`
}

// ToDomainParams converts the request to service parameters.
func (c CreateConversionDTO) ToDomainParams() transaction.CreateConversionParams {
	return transaction.CreateConversionParams{
		Reference:       c.Reference,
		SourceAccountID: c.SourceAccountID,
		TargetAccountID: c.TargetAccountID,
		SourceAmount:    c.SourceAmount,
		Rate:            c.Rate,
	}
}
