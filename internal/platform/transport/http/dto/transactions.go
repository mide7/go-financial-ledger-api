package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
)

type EntryDTO struct {
	AccountID string `json:"account_id" validate:"required,uuid"`
	Type      string `json:"type" validate:"required,oneof=DEBIT CREDIT"`
	Amount    int64  `json:"amount" validate:"required,gt=0"`
}
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

func (d CreateTransactionDTO) ToDomainParams() (transaction.CreateTransactionParams, error) {
	entries := make([]transaction.EntryParams, len(d.Entries))
	for i, e := range d.Entries {
		accID, err := uuid.Parse(e.AccountID)
		if err != nil {
			return transaction.CreateTransactionParams{}, err
		}
		entries[i] = transaction.EntryParams{
			AccountID: accID,
			Type:      transaction.EntryType(e.Type),
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

type GetTransactionDetailsDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}

type ListTransactionsDTO struct {
	Page   int    `form:"page" validate:"omitempty,gte=1"`
	Limit  int    `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Type   string `form:"type" validate:"omitempty,min=1,max=64"`
	Status string `form:"status" validate:"omitempty,oneof=POSTED REVERSED FAILED"`
}

type ReverseTransactionDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}

type GetTransactionEntriesDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}
