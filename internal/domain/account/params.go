package account

import "time"

type CreateAccountParams struct {
	OwnerId  string
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
	ID            string
	AsOfTimestamp time.Time
}

type CreateAccountSnapshotParams struct {
	ID                  string
	TargetEntrySequence int64
	Reason              string
}
