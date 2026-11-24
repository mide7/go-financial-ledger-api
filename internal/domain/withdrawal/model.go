// Package withdrawal models payouts of user funds to an external processor.
//
// A withdrawal first locks the funds with a ledger transaction (for example
// USER_WALLET → USER_LOCKED), then is settled or failed by a second
// transaction once the processor reports back. Not implemented yet: the
// withdrawals table is still drafted, commented out, in the core migration.
package withdrawal

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// Errors returned by the withdrawal service and repository.
var (
	ErrWithdrawalNotFound = errors.New("withdrawal not found")
	ErrNotProcessing      = errors.New("withdrawal is no longer processing")
)

// Status is a withdrawal's lifecycle state; see the Status* constants.
type Status string

// Withdrawal statuses. PROCESSING is the only status that can change.
const (
	StatusProcessing Status = "PROCESSING"
	StatusSettled    Status = "SETTLED"
	StatusFailed     Status = "FAILED"
)

// Withdrawal is a payout request and the ledger transactions that track it.
type Withdrawal struct {
	ID                      uuid.UUID     `json:"id"`
	AccountID               uuid.UUID     `json:"account_id"`
	Amount                  int64         `json:"amount"`
	Currency                currency.Code `json:"currency"`
	LockTransactionID       uuid.UUID     `json:"lock_transaction_id"`
	ResolutionTransactionID uuid.NullUUID `json:"resolution_transaction_id"`
	Processor               string        `json:"processor"`
	ProcessorReference      *string       `json:"processor_reference"`
	Status                  Status        `json:"status"`
	CreatedAt               time.Time     `json:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at"`
	ResolvedAt              *time.Time    `json:"resolved_at"`
}
