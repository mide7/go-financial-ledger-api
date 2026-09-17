package repository

import (
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

func mapToAccountDomain(a *sqlc.Account) *account.Account {
	if a == nil {
		return nil
	}

	return &account.Account{
		ID:        a.ID,
		OwnerID:   a.OwnerID,
		Type:      a.Type,
		Currency:  currency.Code(a.Currency),
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

func mapToTransactionDomain(t *sqlc.Transaction) *transaction.Transaction {
	if t == nil {
		return nil
	}

	return &transaction.Transaction{
		ID:                  t.ID,
		Reference:           t.Reference,
		Type:                t.Type,
		Currency:            t.Currency,
		ParentTransactionID: t.ParentTransactionID,
		Description:         t.Description,
		Metadata:            t.Metadata,
		CreatedAt:           t.CreatedAt,
	}
}
