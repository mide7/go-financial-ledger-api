package account

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrInvalidCurrency  = errors.New("invalid currency code: must be 3-letter ISO code")
	ErrInvalidType      = errors.New("invalid account type")
	ErrInvalidStatus    = errors.New("invalid account status")
	ErrInvalidActorType = errors.New("invalid actor type")
	ErrAccountExists    = errors.New("account already exists for owner, type, and currency")
	ErrDiscrepancyFound = errors.New("reconciliation failure: balance discrepancy detected")
)

type AccountStatus string
type AccountType string
type ActorType string

// Allowed Account Types
const (
	TypeUserWallet              AccountType = "USER_WALLET"
	TypeUserLocked              AccountType = "USER_LOCKED"
	TypeMerchantWallet          AccountType = "MERCHANT_WALLET"
	TypeMerchantPayable         AccountType = "MERCHANT_PAYABLE"
	TypeMerchantReserve         AccountType = "MERCHANT_RESERVE"
	TypePlatformEscrowLiability AccountType = "PLATFORM_ESCROW_LIABILITY"
	TypePlatformFeeRevenue      AccountType = "PLATFORM_FEE_REVENUE"
	TypePlatformCash            AccountType = "PLATFORM_CASH"
	TypePlatformFxSuspense      AccountType = "PLATFORM_FX_SUSPENSE"
	TypePlatformFxGainLoss      AccountType = "PLATFORM_FX_GAIN_LOSS"
)

// Allowed Account Statuses
const (
	StatusActive           AccountStatus = "ACTIVE"
	StatusFullyFrozen      AccountStatus = "FULLY_FROZEN"
	StatusDebitRestricted  AccountStatus = "DEBIT_RESTRICTED"
	StatusCreditRestricted AccountStatus = "CREDIT_RESTRICTED"
	StatusClosed           AccountStatus = "CLOSED"
)

// Allowed Actor Types
const (
	ActorTypeSystem ActorType = "SYSTEM"
	ActorTypeAdmin  ActorType = "ADMIN"
)

type Account struct {
	ID        uuid.UUID     `json:"id"`
	OwnerID   string        `json:"owner_id"`
	Type      AccountType   `json:"type"`
	Currency  currency.Code `json:"currency"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type StatusHistory struct {
	ID             uuid.UUID     `json:"id"`
	AccountID      uuid.UUID     `json:"account_id"`
	PreviousStatus AccountStatus `json:"previous_status"`
	NewStatus      AccountStatus `json:"new_status"`
	Reason         string        `json:"reason"`
	ActorType      ActorType     `json:"actor_type"`
	ActorID        string        `json:"actor_id"`
	CreatedAt      time.Time     `json:"created_at"`
}

type GetAccountBalanceResult struct {
	Currency     string         `json:"currency"`
	Exponent     int16          `json:"exponent"`
	BalanceMinor int64          `json:"balance_minor"` // BIGINT maps to int64
	BalanceMajor pgtype.Numeric `json:"balance_major"` // NUMERIC maps to pgtype.Numeric
}

// Statement represents a time-windowed financial report for an account.
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

type StatementEntry struct {
	ID             string        `json:"id"`
	TransactionID  uuid.UUID     `json:"transaction_id"`
	Type           string        `json:"type"`   // "DEBIT" or "CREDIT"
	Amount         int64         `json:"amount"` // Minor units
	RunningBalance int64         `json:"running_balance"`
	Currency       currency.Code `json:"currency"`
	Exponent       int16         `json:"exponent"`
	Description    string        `json:"description"`
	EntrySequence  int64         `json:"entry_sequence"`
	CreatedAt      time.Time     `json:"created_at"`
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

type DiscrepancyRecord struct{}

type BalanceSnapshot struct{}
