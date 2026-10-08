package account

import (
	"time"

	"github.com/google/uuid"
)

type CreateAccountParams struct {
	OwnerID  string
	Type     string
	Currency string
}

type GetAccountByOwnerTypeCurrencyParams struct {
	OwnerID  string
	Type     string
	Currency string
}

type ListAccountsParams struct {
	Page     int
	Limit    int
	SortBy   string
	OrderBy  string
	Currency string
	Type     string
	OwnerID  string
}

type ReconcileAccountParams struct {
	ID            uuid.UUID
	AsOfTimestamp time.Time
}

type CreateAccountSnapshotParams struct {
	ID                  uuid.UUID
	TargetEntrySequence int64
	Reason              string
}

type GetStatementSummaryParams struct {
	AccountID uuid.UUID
	StartDate time.Time
	EndDate   time.Time
}

type GetStatementEntriesParams struct {
	AccountID uuid.UUID
	StartDate time.Time
	EndDate   time.Time
	Limit     int32
	Cursor    uuid.NullUUID
}

type StatusChangeParams struct {
	AccountID      uuid.UUID
	NewStatus      AccountStatus
	ExpectedStatus *AccountStatus // optimistic check; nil = skip
	Reason         string
	ActorType      ActorType // SYSTEM | ADMIN
	ActorID        string
}

type ListAccountStatusHistoryParams struct {
	AccountID       uuid.UUID
	CursorID        uuid.NullUUID
	CursorCreatedAt *time.Time
	Limit           int32
}
