package transaction

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
)

func debit(amount int64) EntryParams {
	return EntryParams{AccountID: uuid.New(), Type: domain.EntryTypeDebit, Amount: amount}
}

func credit(amount int64) EntryParams {
	return EntryParams{AccountID: uuid.New(), Type: domain.EntryTypeCredit, Amount: amount}
}

func TestCreateTransactionParams_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries []EntryParams
		wantErr error
	}{
		{name: "balanced pair", entries: []EntryParams{debit(100), credit(100)}},
		{name: "balanced multi-leg", entries: []EntryParams{debit(70), debit(30), credit(60), credit(40)}},
		{name: "balanced at int64 max", entries: []EntryParams{debit(math.MaxInt64), credit(math.MaxInt64)}},
		{name: "no entries", entries: nil, wantErr: ErrAtLeastTwoEntries},
		{name: "single entry", entries: []EntryParams{debit(100)}, wantErr: ErrAtLeastTwoEntries},
		{
			name:    "missing account",
			entries: []EntryParams{debit(100), {Type: domain.EntryTypeCredit, Amount: 100}},
			wantErr: ErrInvalidAccountID,
		},
		{name: "zero amount", entries: []EntryParams{debit(0), credit(0)}, wantErr: ErrInvalidEntryAmount},
		{name: "negative amount", entries: []EntryParams{debit(-5), credit(-5)}, wantErr: ErrInvalidEntryAmount},
		{
			name:    "unknown entry type",
			entries: []EntryParams{debit(100), {AccountID: uuid.New(), Type: "SIDEWAYS", Amount: 100}},
			wantErr: ErrInvalidEntryType,
		},
		{
			name:    "debits exceed credits",
			entries: []EntryParams{debit(101), credit(100)},
			wantErr: ErrUnbalancedTransaction,
		},
		{name: "credits only", entries: []EntryParams{credit(100), credit(100)}, wantErr: ErrUnbalancedTransaction},
		{
			// MaxInt64 + MaxInt64 + 2 == 2^64, which wraps to 0 in int64 arithmetic.
			name:    "credits only, sum overflows to zero",
			entries: []EntryParams{credit(math.MaxInt64), credit(math.MaxInt64), credit(2)},
			wantErr: ErrAmountOverflow,
		},
		{
			name:    "balanced but each side overflows",
			entries: []EntryParams{debit(math.MaxInt64), debit(1), credit(math.MaxInt64), credit(1)},
			wantErr: ErrAmountOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CreateTransactionParams{Entries: tt.entries}.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// FuzzCreateTransactionParams_Validate checks Validate against an exact
// arbitrary-precision reference: a transaction of valid legs is accepted if and
// only if debits equal credits and neither side's total overflows int64.
func FuzzCreateTransactionParams_Validate(f *testing.F) {
	f.Add(int64(100), int64(100), int64(0), true, false, false)
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), int64(2), false, false, false)
	f.Add(int64(70), int64(30), int64(100), true, true, false)

	f.Fuzz(func(t *testing.T, a, b, c int64, aDebit, bDebit, cDebit bool) {
		if a <= 0 || b <= 0 || c <= 0 {
			t.Skip("amount validity is covered by the table test")
		}

		entries := make([]EntryParams, 0, 3)
		debits, credits := new(big.Int), new(big.Int)

		for _, leg := range []struct {
			amount  int64
			isDebit bool
		}{{a, aDebit}, {b, bDebit}, {c, cDebit}} {
			if leg.isDebit {
				entries = append(entries, debit(leg.amount))
				debits.Add(debits, big.NewInt(leg.amount))
			} else {
				entries = append(entries, credit(leg.amount))
				credits.Add(credits, big.NewInt(leg.amount))
			}
		}

		maxInt64 := big.NewInt(math.MaxInt64)
		overflows := debits.Cmp(maxInt64) > 0 || credits.Cmp(maxInt64) > 0
		wantValid := !overflows && debits.Cmp(credits) == 0

		err := CreateTransactionParams{Entries: entries}.Validate()
		if (err == nil) != wantValid {
			t.Fatalf("Validate() error = %v; debits %s, credits %s, want valid=%v", err, debits, credits, wantValid)
		}
	})
}
