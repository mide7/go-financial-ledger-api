package account

import (
	"time"

	"github.com/google/uuid"
)

// CreateAccountParams are the inputs for opening an account.
type CreateAccountParams struct {
	OwnerID  string
	Type     string
	Currency string
}

// GetAccountByOwnerTypeCurrencyParams identify an account by its unique owner, type and currency.
type GetAccountByOwnerTypeCurrencyParams struct {
	OwnerID  string
	Type     string
	Currency string
}

// Sort fields and orders accepted by ListAccounts.
const (
	SortByCreatedAt = "created_at"
	SortByType      = "type"
	SortByCurrency  = "currency"
	SortOrderAsc    = "asc"
	SortOrderDesc   = "desc"
)

// ListAccountsParams filter, sort and offset-paginate an account listing.
// Empty filters match everything. SortBy defaults to created_at and OrderBy
// (asc or desc) to desc; ties are broken by id, newest first.
type ListAccountsParams struct {
	Page     int
	Limit    int
	SortBy   string
	OrderBy  string
	Currency string
	Type     string
	OwnerID  string
}

// ListDiscrepanciesParams cursor-paginate the unresolved discrepancies,
// newest first. Cursor is the id of the last record on the previous page.
type ListDiscrepanciesParams struct {
	Cursor uuid.NullUUID
	Limit  int32
}

// CreateAccountSnapshotParams request a balance snapshot covering every entry
// posted so far. Reason is logged for audit; there is no column to store it.
type CreateAccountSnapshotParams struct {
	ID     uuid.UUID
	Reason string
}

// GetStatementSummaryParams select an account and date range for a statement summary.
type GetStatementSummaryParams struct {
	AccountID uuid.UUID
	StartDate time.Time
	EndDate   time.Time
}

// GetStatementEntriesParams select an account, date range and cursor page of statement entries.
type GetStatementEntriesParams struct {
	AccountID uuid.UUID
	StartDate time.Time
	EndDate   time.Time
	Limit     int32
	Cursor    uuid.NullUUID
}

// StatusChangeParams describe a requested account status change.
type StatusChangeParams struct {
	AccountID      uuid.UUID
	NewStatus      Status
	ExpectedStatus *Status // optimistic check; nil = skip
	Reason         string
	ActorType      ActorType // SYSTEM | ADMIN
	ActorID        string
}

// ListAccountStatusHistoryParams select a cursor page of an account's status history.
type ListAccountStatusHistoryParams struct {
	AccountID       uuid.UUID
	CursorID        uuid.NullUUID
	CursorCreatedAt *time.Time
	Limit           int32
}

// CreateDiscrepancyRecordParams hold the reconciliation figures to record as a discrepancy.
type CreateDiscrepancyRecordParams struct {
	AccountID            uuid.UUID
	RawLedgerSum         int64
	SnapshotBalance      int64
	CalculatedTotal      int64
	DiscrepancyAmount    int64
	LastSnapshotSequence int64
	CurrentMaxSequence   int64
	// Currency currency.Code `json:"currency"`
	// Exponent int16 `json:"exponent"`
}
