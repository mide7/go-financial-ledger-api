package transaction

import (
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

type EntryParams struct {
	AccountID uuid.UUID
	Type      EntryType
	Amount    int64
}

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

func (p CreateTransactionParams) Validate() error {
	if len(p.Entries) < 2 {
		return ErrAtLeastTwoEntries
	}

	var netBalance int64
	for _, entry := range p.Entries {
		if entry.AccountID == uuid.Nil {
			return ErrInvalidAccountID
		}
		if entry.Amount <= 0 {
			return ErrInvalidEntryAmount
		}
		if !entry.Type.IsValid() {
			return ErrInvalidEntryType
		}

		if entry.Type == EntryTypeDebit {
			netBalance -= entry.Amount
		}

		if entry.Type == EntryTypeCredit {
			netBalance += entry.Amount
		}

	}

	if netBalance != 0 {
		return ErrUnbalancedTransaction
	}

	return nil
}

type ListTransactionsParams struct {
	Page   int
	Limit  int
	Type   string
	Status string
}
