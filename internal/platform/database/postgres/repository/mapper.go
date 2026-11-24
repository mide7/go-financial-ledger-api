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
		Type:      account.Type(a.Type),
		Currency:  currency.Code(a.Currency),
		Status:    account.Status(a.Status),
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

func mapToAccountDomainList(as []sqlc.Account) []account.Account {
	if len(as) == 0 {
		return nil
	}

	accounts := make([]account.Account, len(as))
	for i, a := range as {
		accounts[i] = *mapToAccountDomain(&a)
	}

	return accounts
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

func mapToDiscrepancyDomain(d *sqlc.DiscrepancyRecord) *account.DiscrepancyRecord {
	record := &account.DiscrepancyRecord{
		ID:                   d.ID,
		AccountID:            d.AccountID,
		RawLedgerSum:         d.RawLedgerSum,
		SnapshotBalance:      d.SnapshotBalance,
		CalculatedTotal:      d.CalculatedTotal,
		DiscrepancyAmount:    d.DiscrepancyAmount,
		LastSnapshotSequence: d.LastSnapshotSequence,
		CurrentMaxSequence:   d.CurrentMaxSequence,
		DetectedAt:           d.DetectedAt,
	}

	if d.ResolvedAt.Valid {
		record.ResolvedAt = &d.ResolvedAt.Time
	}

	if d.ResolutionNotes.Valid {
		record.ResolutionNotes = &d.ResolutionNotes.String
	}

	return record
}

func mapToSnapshotDomain(s *sqlc.BalanceSnapshot) *account.BalanceSnapshot {
	return &account.BalanceSnapshot{
		ID:                s.ID,
		AccountID:         s.AccountID,
		Balance:           s.Balance,
		LastEntrySequence: s.LastEntrySequence,
		CreatedAt:         s.CreatedAt,
	}
}

func mapStringPointerToAccountStatusPointer(s *string) *account.Status {
	if s == nil {
		return nil
	}

	status := account.Status(*s)

	return &status
}
