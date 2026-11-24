// Package transaction models double-entry ledger transactions and their entries.
package transaction

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
)

// Errors returned by the transaction service and repository.
var (
	ErrDuplicateReference    = errors.New("duplicate reference detect")
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrInvalidCurrency       = errors.New("invalid currency code: must be 3-letter ISO code")
	ErrAtLeastTwoEntries     = errors.New("transaction must contain at least two entries")
	ErrUnbalancedTransaction = errors.New("debits and credits must balance to zero")
	ErrInvalidEntryType      = errors.New("invalid entry type")
	ErrInvalidEntryAmount    = errors.New("entry amount must be greater than zero")
	ErrInvalidAccountID      = errors.New("account ID cannot be empty")
	ErrAmountOverflow        = errors.New("total debits or credits exceed the maximum amount")
	ErrAlreadyReversed       = errors.New("transaction has already been reversed")
	ErrCannotReverseReversal = errors.New("a reversal cannot itself be reversed")
	ErrAccountNotPostable    = errors.New("account status does not allow this entry")
	ErrCurrencyMismatch      = errors.New("entry account currency does not match the transaction currency")
	ErrInvalidDateRange      = errors.New("start date must be before end date")
	ErrEntryAccountNotFound  = errors.New("entry account not found")
	ErrInvalidMetadata       = errors.New("metadata must be a JSON object")
	ErrReservedType          = errors.New("transaction type REVERSAL is reserved for reversals")
)

// TypeReversal is the type of a transaction posted by ReverseTransaction. Its
// parent_transaction_id is the transaction it reverses, and its description is
// the reason given. CreateTransaction rejects it, so only ReverseTransaction
// can post one.
const TypeReversal = "REVERSAL"

// Entry is one debit or credit leg of a transaction, in minor units.
type Entry struct {
	ID            uuid.UUID        `json:"id"`
	TransactionID uuid.UUID        `json:"transaction_id"`
	AccountID     uuid.UUID        `json:"account_id"`
	Type          domain.EntryType `json:"type"`
	Amount        int64            `json:"amount"`
	EntrySequence int64            `json:"entry_sequence"`
	CreatedAt     time.Time        `json:"created_at"`
}

// Transaction is a balanced set of entries posted together under a unique reference.
type Transaction struct {
	ID                  uuid.UUID       `json:"id"`
	Reference           string          `json:"reference"`
	Type                string          `json:"type"`
	Currency            string          `json:"currency"`
	ParentTransactionID uuid.NullUUID   `json:"parent_transaction_id"`
	Description         string          `json:"description"`
	Metadata            json.RawMessage `json:"metadata"`
	TransactionDate     time.Time       `json:"transaction_date"`
	CreatedAt           time.Time       `json:"created_at"`
	Entries             []Entry         `json:"entries,omitempty"` // omitted in listings
}

// Conversion is a currency exchange posted as two transactions, one per
// currency, that meet in the PLATFORM_FX_SUSPENSE accounts.
type Conversion struct {
	Reference string      `json:"reference"`
	Rate      string      `json:"rate"` // target units per source unit, as a decimal string
	Source    Transaction `json:"source"`
	Target    Transaction `json:"target"`
}
