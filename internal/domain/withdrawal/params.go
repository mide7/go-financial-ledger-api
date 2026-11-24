package withdrawal

import "github.com/google/uuid"

// CreateWithdrawalParams request a payout of Amount minor units from AccountID.
// Reference is the unique reference of the lock transaction.
type CreateWithdrawalParams struct {
	AccountID uuid.UUID
	Amount    int64
	Processor string
	Reference string
}

// ListWithdrawalsParams filter and cursor-paginate withdrawals, newest first.
type ListWithdrawalsParams struct {
	AccountID uuid.NullUUID
	Status    Status
	Cursor    uuid.NullUUID
	Limit     int32
}

// SettleWithdrawalParams record that the processor paid out withdrawal ID.
type SettleWithdrawalParams struct {
	ID                 uuid.UUID
	ProcessorReference string
}

// FailWithdrawalParams record that the payout failed; the locked funds are released.
type FailWithdrawalParams struct {
	ID     uuid.UUID
	Reason string
}
