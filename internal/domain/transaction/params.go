package transaction

import (
	"bytes"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// EntryParams describe one leg of a transaction to create.
type EntryParams struct {
	AccountID uuid.UUID
	Type      domain.EntryType
	Amount    int64
}

// CreateTransactionParams are the inputs for posting a transaction.
type CreateTransactionParams struct {
	Reference           string
	Type                string
	Currency            currency.Code
	ParentTransactionID uuid.NullUUID

	Description     string
	Metadata        json.RawMessage // a JSON object; empty or null means {}
	TransactionDate time.Time
	Entries         []EntryParams
}

// Validate checks the double-entry invariants: at least two entries, each with an
// account and a positive amount, and debits equal to credits.
//
// Each side is totalled separately and must fit in int64. A single running net
// would wrap on overflow, letting an unbalanced set of large entries net to zero.
func (p CreateTransactionParams) Validate() error {
	if !isObjectOrEmpty(p.Metadata) {
		return ErrInvalidMetadata
	}

	if len(p.Entries) < 2 {
		return ErrAtLeastTwoEntries
	}

	var debits, credits int64

	for _, entry := range p.Entries {
		if entry.AccountID == uuid.Nil {
			return ErrInvalidAccountID
		}

		if entry.Amount <= 0 {
			return ErrInvalidEntryAmount
		}

		total := &credits

		switch entry.Type {
		case domain.EntryTypeDebit:
			total = &debits
		case domain.EntryTypeCredit:
		default:
			return ErrInvalidEntryType
		}

		if *total > math.MaxInt64-entry.Amount {
			return ErrAmountOverflow
		}

		*total += entry.Amount
	}

	if debits != credits {
		return ErrUnbalancedTransaction
	}

	return nil
}

// emptyObject is the metadata stored when a request omits it.
var emptyObject = json.RawMessage("{}")

// isObjectOrEmpty reports whether m is a JSON object, or empty or null.
func isObjectOrEmpty(m json.RawMessage) bool {
	m = bytes.TrimSpace(m)
	if len(m) == 0 || bytes.Equal(m, []byte("null")) {
		return true
	}

	return m[0] == '{' && json.Valid(m)
}

// metadataOrEmpty returns the metadata to store: emptyObject when the request
// omitted it or sent null.
func (p CreateTransactionParams) metadataOrEmpty() json.RawMessage {
	if m := bytes.TrimSpace(p.Metadata); len(m) > 0 && !bytes.Equal(m, []byte("null")) {
		return m
	}

	return emptyObject
}

// ListTransactionsParams filter and cursor-paginate a transaction listing,
// newest first. Zero values mean "no filter". StartDate is inclusive and
// EndDate exclusive.
type ListTransactionsParams struct {
	AccountID uuid.NullUUID
	Reference string
	Type      string
	Currency  currency.Code
	StartDate time.Time
	EndDate   time.Time
	Cursor    uuid.NullUUID // id of the last transaction on the previous page
	Limit     int32
}

// ReverseTransactionParams request a reversal of transaction ID. Reference is
// the reversal's own unique reference, so a retried request can't post twice.
type ReverseTransactionParams struct {
	ID        uuid.UUID
	Reference string
	Reason    string
}

// CreateConversionParams request converting SourceAmount from the source
// account's currency into the target account's currency at Rate.
type CreateConversionParams struct {
	Reference       string
	SourceAccountID uuid.UUID
	TargetAccountID uuid.UUID
	SourceAmount    int64  // minor units of the source currency
	Rate            string // target units per source unit, as a decimal string
}
