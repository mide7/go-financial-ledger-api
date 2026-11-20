package transaction

import (
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
	Metadata        []byte
	TransactionDate time.Time
	Entries         []EntryParams
}

// Validate checks the double-entry invariants: at least two entries, each with an
// account and a positive amount, and debits equal to credits.
//
// Each side is totalled separately and must fit in int64. A single running net
// would wrap on overflow, letting an unbalanced set of large entries net to zero.
func (p CreateTransactionParams) Validate() error {
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

// ListTransactionsParams filter and paginate a transaction listing.
type ListTransactionsParams struct {
	Page   int
	Limit  int
	Type   string
	Status string
}
