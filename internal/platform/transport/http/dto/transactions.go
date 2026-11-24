package dto

import (
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
	Reference           string     `json:"reference" validate:"required,min=1,max=128"`
	Type                string     `json:"type" validate:"required,min=1,max=64"`
	Currency            string     `json:"currency" validate:"required,len=3"`
	ParentTransactionID *string    `json:"parent_transaction_id" validate:"omitempty,uuid"`
	Description         string     `json:"description" validate:"required,max=256"`
	Metadata            []byte     `json:"metadata" validate:"omitempty"`
	TransactionDate     time.Time  `json:"transaction_date" validate:"required"`
	Entries             []EntryDTO `json:"entries" validate:"required,min=2,dive"`
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
	ID string `form:"id" validate:"required,uuid"`
}

// ListTransactionsDTO holds the query parameters of GET /transactions.
type ListTransactionsDTO struct {
	Page   int    `form:"page" validate:"omitempty,gte=1"`
	Limit  int    `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Type   string `form:"type" validate:"omitempty,min=1,max=64"`
	Status string `form:"status" validate:"omitempty,oneof=POSTED REVERSED FAILED"`
}

// ToDomainParams converts the request to service parameters.
func (l ListTransactionsDTO) ToDomainParams() transaction.ListTransactionsParams {
	return transaction.ListTransactionsParams{
		Page:   l.Page,
		Limit:  l.Limit,
		Type:   l.Type,
		Status: l.Status,
	}
}

// ReverseTransactionDTO holds the path parameters of POST /transactions/{id}/reverse.
type ReverseTransactionDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}

// GetTransactionEntriesDTO holds the path parameters for listing a transaction's entries.
type GetTransactionEntriesDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}
