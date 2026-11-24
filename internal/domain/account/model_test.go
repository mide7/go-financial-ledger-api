package account

import (
	"errors"
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/domain"
)

func TestType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Type
		want bool
	}{
		{name: "credit-normal liability", in: TypeUserWallet, want: true},
		{name: "debit-normal asset", in: TypePlatformCash, want: true},
		{name: "lowercase", in: "user_wallet", want: false},
		{name: "empty", in: "", want: false},
		{name: "unknown", in: "SAVINGS", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.in.IsValid(); got != tt.want {
				t.Errorf("Type(%q).IsValid() = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestAllTypes guards the seeder, which creates one system account per entry:
// every entry must be valid and appear once.
func TestAllTypes(t *testing.T) {
	t.Parallel()

	seen := make(map[Type]bool, len(AllTypes))

	for _, typ := range AllTypes {
		if !typ.IsValid() {
			t.Errorf("AllTypes contains invalid type %q", typ)
		}

		if seen[typ] {
			t.Errorf("AllTypes lists %q more than once", typ)
		}

		seen[typ] = true
	}

	if len(AllTypes) != 10 {
		t.Errorf("len(AllTypes) = %d, want 10; update this test and the seeder's account_types when adding a type",
			len(AllTypes),
		)
	}
}

func TestStatus_AllowsEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status        Status
		debit, credit bool
	}{
		{status: StatusActive, debit: true, credit: true},
		{status: StatusDebitRestricted, debit: false, credit: true},
		{status: StatusCreditRestricted, debit: true, credit: false},
		{status: StatusFullyFrozen, debit: false, credit: false},
		{status: StatusClosed, debit: false, credit: false},
		{status: "BOGUS", debit: false, credit: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()

			if got := tt.status.AllowsEntry(domain.EntryTypeDebit); got != tt.debit {
				t.Errorf("AllowsEntry(DEBIT) = %v, want %v", got, tt.debit)
			}

			if got := tt.status.AllowsEntry(domain.EntryTypeCredit); got != tt.credit {
				t.Errorf("AllowsEntry(CREDIT) = %v, want %v", got, tt.credit)
			}
		})
	}
}

func TestStatus_AllowsReversal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status Status
		want   bool
	}{
		{status: StatusActive, want: true},
		{status: StatusDebitRestricted, want: true},
		{status: StatusCreditRestricted, want: true},
		{status: StatusFullyFrozen, want: true},
		{status: StatusClosed, want: false},
		{status: "BOGUS", want: false},
	}

	for _, tt := range tests {
		if got := tt.status.AllowsReversal(); got != tt.want {
			t.Errorf("%s.AllowsReversal() = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestStatus_CheckTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		from, to Status
		want     error
	}{
		{name: "active to frozen", from: StatusActive, to: StatusFullyFrozen},
		{name: "frozen back to active", from: StatusFullyFrozen, to: StatusActive},
		{name: "active to closed", from: StatusActive, to: StatusClosed},
		{name: "same status", from: StatusDebitRestricted, to: StatusDebitRestricted, want: ErrStatusUnchanged},
		{name: "closed is final", from: StatusClosed, to: StatusActive, want: ErrAccountClosed},
		{name: "closed to closed", from: StatusClosed, to: StatusClosed, want: ErrAccountClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.from.CheckTransition(tt.to); !errors.Is(err, tt.want) {
				t.Errorf("%s.CheckTransition(%s) = %v, want %v", tt.from, tt.to, err, tt.want)
			}
		})
	}
}
