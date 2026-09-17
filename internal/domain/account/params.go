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

type GetAccountDetailsParams struct {
	ID string
}

type ListAccountsParams struct {
	Page     int
	Limit    int
	Currency string
	Type     string
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
