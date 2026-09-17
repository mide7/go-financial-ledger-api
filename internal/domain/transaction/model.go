package transaction

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDuplicateReference    = errors.New("duplicate reference detect")
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrInvalidCurrency       = errors.New("invalid currency code: must be 3-letter ISO code")
	ErrAtLeastTwoEntries     = errors.New("transaction must contain at least two entries")
	ErrUnbalancedTransaction = errors.New("debits and credits must balance to zero")
	ErrInvalidEntryType      = errors.New("invalid entry type")
	ErrInvalidEntryAmount    = errors.New("entry amount must be greater than zero")
	ErrInvalidAccountID      = errors.New("account ID cannot be empty")
)

type EntryType string

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

func (t EntryType) IsValid() bool {
	switch t {
	case EntryTypeDebit, EntryTypeCredit:
		return true
	default:
		return false
	}
}

type Entry struct {
	ID            uuid.UUID `json:"id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	AccountID     uuid.UUID `json:"account_id"`
	Type          EntryType `json:"type"`
	Amount        int64     `json:"amount"`
	EntrySequence int64     `json:"entry_sequence"`
	CreatedAt     time.Time `json:"created_at"`
}

type Transaction struct {
	ID                  uuid.UUID     `json:"id"`
	Reference           string        `json:"reference"`
	Type                string        `json:"type"`
	Currency            string        `json:"currency"`
	ParentTransactionID uuid.NullUUID `json:"parent_transaction_id"`
	Description         string        `json:"description"`
	Metadata            []byte        `json:"metadata"`
	CreatedAt           time.Time     `json:"created_at"`
	Entries             []Entry       `json:"entries"`
}
