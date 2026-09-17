package account

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrInvalidCurrency  = errors.New("invalid currency code: must be 3-letter ISO code")
	ErrInvalidType      = errors.New("invalid account type")
	ErrAccountExists    = errors.New("account already exists for owner, type, and currency")
	ErrDiscrepancyFound = errors.New("reconciliation failure: balance discrepancy detected")
)

// Allowed Account Types
const (
	TypeUserWallet              = "USER_WALLET"
	TypePlatformEscrowLiability = "PLATFORM_ESCROW_LIABILITY"
	TypePlatformFeeRevenue      = "PLATFORM_FEE_REVENUE"
	TypeMerchantPayable         = "MERCHANT_PAYABLE"
)

type Account struct {
	ID        uuid.UUID     `json:"id"`
	OwnerID   string        `json:"owner_id"`
	Type      string        `json:"type"`
	Currency  currency.Code `json:"currency"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Statement represents a time-windowed financial report for an account.
type Statement struct {
	AccountID      uuid.UUID        `json:"account_id"`
	Currency       currency.Code    `json:"currency"`
	StartDate      time.Time        `json:"start_date"`
	EndDate        time.Time        `json:"end_date"`
	OpeningBalance int64            `json:"opening_balance"`
	ClosingBalance int64            `json:"closing_balance"`
	TotalDebits    int64            `json:"total_debits"`
	TotalCredits   int64            `json:"total_credits"`
	Entries        []StatementEntry `json:"entries"`
}

type StatementEntry struct {
	ID            string    `json:"id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	Type          string    `json:"type"`   // "DEBIT" or "CREDIT"
	Amount        int64     `json:"amount"` // Minor units
	EntrySequence int64     `json:"entry_sequence"`
	CreatedAt     time.Time `json:"created_at"`
}

// ReconciliationResult detail audit output.
type ReconciliationResult struct {
	AccountID            uuid.UUID     `json:"account_id"`
	Status               string        `json:"status"` // "RECONCILED" or "DISCREPANCY_DETECTED"
	Currency             currency.Code `json:"currency"`
	RawLedgerSum         int64         `json:"raw_ledger_sum"`
	SnapshotBalance      int64         `json:"snapshot_balance"`
	DeltaSinceSnapshot   int64         `json:"delta_since_snapshot"`
	CalculatedTotal      int64         `json:"calculated_total"`
	DiscrepancyAmount    int64         `json:"discrepancy_amount"`
	LastSnapshotSequence int64         `json:"last_snapshot_sequence"`
	CurrentMaxSequence   int64         `json:"current_max_sequence"`
	ReconciledAt         time.Time     `json:"reconciled_at"`
}
