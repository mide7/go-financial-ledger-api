// Package account models ledger accounts, their balances, statements and reconciliation.
package account

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// Errors returned by the account service and repository.
var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrInvalidCurrency     = errors.New("invalid currency code: must be 3-letter ISO code")
	ErrInvalidType         = errors.New("invalid account type")
	ErrInvalidStatus       = errors.New("invalid account status")
	ErrInvalidActorType    = errors.New("invalid actor type")
	ErrAccountExists       = errors.New("account already exists for owner, type, and currency")
	ErrDiscrepancyFound    = errors.New("reconciliation failure: balance discrepancy detected")
	ErrSnapshotNotFound    = errors.New("no balance snapshot exists for account")
	ErrSnapshotExists      = errors.New("no new entries since the last balance snapshot")
	ErrStatusConflict      = errors.New("account is not in the expected status")
	ErrStatusUnchanged     = errors.New("account is already in that status")
	ErrAccountClosed       = errors.New("account is closed; its status can't change")
	ErrDiscrepancyNotFound = errors.New("discrepancy not found or already resolved")
)

// Type is an account's role in the chart of accounts; see the Type* constants.
type Type string

// Allowed Account Types.
const (
	TypeUserWallet              Type = "USER_WALLET"
	TypeUserLocked              Type = "USER_LOCKED"
	TypeMerchantWallet          Type = "MERCHANT_WALLET"
	TypeMerchantPayable         Type = "MERCHANT_PAYABLE"
	TypeMerchantReserve         Type = "MERCHANT_RESERVE"
	TypePlatformEscrowLiability Type = "PLATFORM_ESCROW_LIABILITY"
	TypePlatformFeeRevenue      Type = "PLATFORM_FEE_REVENUE"
	TypePlatformCash            Type = "PLATFORM_CASH"
	TypePlatformFxSuspense      Type = "PLATFORM_FX_SUSPENSE"
	TypePlatformFxGainLoss      Type = "PLATFORM_FX_GAIN_LOSS"
)

// AllTypes lists every account type, in chart-of-accounts order.
var AllTypes = []Type{
	TypeUserWallet, TypeUserLocked, TypeMerchantWallet, TypeMerchantPayable, TypeMerchantReserve,
	TypePlatformEscrowLiability, TypePlatformFeeRevenue, TypePlatformCash,
	TypePlatformFxSuspense, TypePlatformFxGainLoss,
}

// IsValid reports whether t is one of the Type* constants.
func (t Type) IsValid() bool {
	switch t {
	case TypeUserWallet, TypeUserLocked, TypeMerchantWallet, TypeMerchantPayable, TypeMerchantReserve,
		TypePlatformEscrowLiability, TypePlatformFeeRevenue, TypePlatformCash,
		TypePlatformFxSuspense, TypePlatformFxGainLoss:
		return true
	default:
		return false
	}
}

// Status is an account's lifecycle state; see the Status* constants.
type Status string

// Allowed Account Statuses.
const (
	StatusActive           Status = "ACTIVE"
	StatusFullyFrozen      Status = "FULLY_FROZEN"
	StatusDebitRestricted  Status = "DEBIT_RESTRICTED"
	StatusCreditRestricted Status = "CREDIT_RESTRICTED"
	StatusClosed           Status = "CLOSED"
)

func (s Status) String() string {
	return string(s)
}

// IsValid reports whether s is one of the Status* constants.
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusFullyFrozen, StatusDebitRestricted, StatusCreditRestricted, StatusClosed:
		return true
	default:
		return false
	}
}

// ActorType identifies who changed an account's status; see the ActorType* constants.
type ActorType string

// Allowed Actor Types.
const (
	ActorTypeSystem ActorType = "SYSTEM"
	ActorTypeAdmin  ActorType = "ADMIN"
)

func (at ActorType) String() string {
	return string(at)
}

// IsValid reports whether at is one of the ActorType* constants.
func (at ActorType) IsValid() bool {
	return at == ActorTypeSystem || at == ActorTypeAdmin
}

// Reconciliation outcomes reported in ReconciliationResult.Status.
const (
	ReconciliationStatusReconciled  = "RECONCILED"
	ReconciliationStatusDiscrepancy = "DISCREPANCY_DETECTED"
)

// Account is a single-currency ledger account belonging to an owner.
type Account struct {
	ID        uuid.UUID     `json:"id"`
	OwnerID   string        `json:"owner_id"`
	Type      Type          `json:"type"`
	Currency  currency.Code `json:"currency"`
	Status    Status        `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// AllowsEntry reports whether an account in status s accepts an entry of type t.
// Restrictions are in ledger terms: DEBIT_RESTRICTED refuses debit entries and
// CREDIT_RESTRICTED refuses credit entries, whatever the account's normal
// balance. FULLY_FROZEN and CLOSED refuse both.
func (s Status) AllowsEntry(t domain.EntryType) bool {
	switch s {
	case StatusActive:
		return true
	case StatusDebitRestricted:
		return t == domain.EntryTypeCredit
	case StatusCreditRestricted:
		return t == domain.EntryTypeDebit
	case StatusFullyFrozen, StatusClosed:
		return false
	default:
		return false
	}
}

// AllowsReversal reports whether an account in status s accepts the entries
// of a reversal. A reversal corrects an earlier posting, so restrictions and
// freezes don't block it; only CLOSED (and an unknown status) does.
func (s Status) AllowsReversal() bool {
	return s.IsValid() && s != StatusClosed
}

// CheckTransition returns nil when an account may move from status s to next:
// ErrStatusUnchanged when next equals s, ErrAccountClosed when s is CLOSED
// (closing is final).
func (s Status) CheckTransition(next Status) error {
	if s == StatusClosed {
		return ErrAccountClosed
	}

	if s == next {
		return ErrStatusUnchanged
	}

	return nil
}

// StatusHistory records one change of an account's status and who made it.
type StatusHistory struct {
	ID             uuid.UUID `json:"id"`
	AccountID      uuid.UUID `json:"account_id"`
	PreviousStatus *Status   `json:"previous_status"`
	NewStatus      Status    `json:"new_status"`
	Reason         string    `json:"reason"`
	ActorType      ActorType `json:"actor_type"`
	ActorID        string    `json:"actor_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// Balance sign convention: every balance in this package is positive in the
// account's normal-balance direction (account_types.normal_balance). Entries on
// the normal side add to it and entries on the other side subtract, so a funded
// USER_WALLET (credit-normal) and a funded PLATFORM_CASH (debit-normal) are both
// positive.

// GetAccountBalanceResult is an account's current balance in minor and major
// units, signed by the balance sign convention above.
type GetAccountBalanceResult struct {
	Currency     string `json:"currency"`
	Exponent     int16  `json:"exponent"`
	BalanceMinor int64  `json:"balance_minor"`
	BalanceMajor string `json:"balance_major"` // exact decimal string, e.g. "1234.50"
}

// StatementSummary is an account's opening and closing balance and totals over a
// date range. Balances follow the sign convention above; TotalDebits and
// TotalCredits are unsigned sums.
type StatementSummary struct {
	AccountID      uuid.UUID     `json:"account_id"`
	Currency       currency.Code `json:"currency"`
	Exponent       int16         `json:"exponent"`
	StartDate      time.Time     `json:"start_date"`
	EndDate        time.Time     `json:"end_date"`
	OpeningBalance int64         `json:"opening_balance"`
	ClosingBalance int64         `json:"closing_balance"`
	TotalDebits    int64         `json:"total_debits"`
	TotalCredits   int64         `json:"total_credits"`
}

// StatementEntry is one posted entry on an account statement, with the running balance after it.
type StatementEntry struct {
	ID             uuid.UUID        `json:"id"`
	TransactionID  uuid.UUID        `json:"transaction_id"`
	Type           domain.EntryType `json:"type"`            // "DEBIT" or "CREDIT"
	Amount         int64            `json:"amount"`          // Minor units
	RunningBalance int64            `json:"running_balance"` // after this entry, normal-balance positive
	Currency       currency.Code    `json:"currency"`
	Exponent       int16            `json:"exponent"`
	Description    string           `json:"description"`
	EntrySequence  int64            `json:"entry_sequence"`
	CreatedAt      time.Time        `json:"created_at"`
}

// ReconciliationResult detail audit output.
type ReconciliationResult struct {
	AccountID            uuid.UUID     `json:"account_id"`
	Status               string        `json:"status"` // "RECONCILED" or "DISCREPANCY_DETECTED"
	Currency             currency.Code `json:"currency"`
	Exponent             int16         `json:"exponent"`
	RawLedgerSum         int64         `json:"raw_ledger_sum"`
	SnapshotBalance      int64         `json:"snapshot_balance"`
	DeltaSinceSnapshot   int64         `json:"delta_since_snapshot"`
	CalculatedTotal      int64         `json:"calculated_total"`
	DiscrepancyAmount    int64         `json:"discrepancy_amount"`
	LastSnapshotSequence int64         `json:"last_snapshot_sequence"`
	CurrentMaxSequence   int64         `json:"current_max_sequence"`
	ReconciledAt         time.Time     `json:"reconciled_at"`
}

// DiscrepancyRecord is a persisted reconciliation failure, open until resolved.
type DiscrepancyRecord struct {
	ID                   uuid.UUID  `json:"id"`
	AccountID            uuid.UUID  `json:"account_id"`
	RawLedgerSum         int64      `json:"raw_ledger_sum"`
	SnapshotBalance      int64      `json:"snapshot_balance"`
	CalculatedTotal      int64      `json:"calculated_total"`
	DiscrepancyAmount    int64      `json:"discrepancy_amount"`
	LastSnapshotSequence int64      `json:"last_snapshot_sequence"`
	CurrentMaxSequence   int64      `json:"current_max_sequence"`
	DetectedAt           time.Time  `json:"detected_at"`
	ResolvedAt           *time.Time `json:"resolved_at,omitempty"`
	ResolutionNotes      *string    `json:"resolution_notes,omitempty"`
}

// BalanceSnapshot is a checkpointed account balance covering every entry up to
// and including LastEntrySequence.
type BalanceSnapshot struct {
	ID                uuid.UUID `json:"id"`
	AccountID         uuid.UUID `json:"account_id"`
	Balance           int64     `json:"balance"` // normal-balance positive
	LastEntrySequence int64     `json:"last_entry_sequence"`
	CreatedAt         time.Time `json:"created_at"`
}

// ReconciliationRun summarises one ReconcileAllAccounts pass.
type ReconciliationRun struct {
	AccountsChecked    int       `json:"accounts_checked"`
	DiscrepanciesFound int       `json:"discrepancies_found"`
	StartedAt          time.Time `json:"started_at"`
	FinishedAt         time.Time `json:"finished_at"`
}
