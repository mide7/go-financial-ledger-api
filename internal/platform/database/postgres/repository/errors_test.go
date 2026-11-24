package repository

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
)

var errDomain = errors.New("domain not found")

func TestNotFound(t *testing.T) {
	t.Parallel()

	other := errors.New("connection reset")

	tests := []struct {
		name string
		in   error
		want error
	}{
		{name: "no rows becomes domain error", in: pgx.ErrNoRows, want: errDomain},
		{name: "wrapped no rows becomes domain error", in: fmt.Errorf("query: %w", pgx.ErrNoRows), want: errDomain},
		{name: "other errors pass through", in: other, want: other},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := notFound(tt.in, errDomain); !errors.Is(got, tt.want) {
				t.Errorf("notFound(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   error
		want bool
	}{
		{name: "unique violation", in: &pgconn.PgError{Code: pgErrUniqueViolation}, want: true},
		{
			name: "wrapped unique violation",
			in:   fmt.Errorf("insert: %w", &pgconn.PgError{Code: pgErrUniqueViolation}),
			want: true,
		},
		{name: "foreign key violation", in: &pgconn.PgError{Code: "23503"}, want: false},
		{name: "non-postgres error", in: errors.New("boom"), want: false},
		{name: "nil", in: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isUniqueViolation(tt.in); got != tt.want {
				t.Errorf("isUniqueViolation(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestPageOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		page, limit int
		want        int32
		wantErr     error
	}{
		{name: "first page starts at zero", page: 1, limit: 20, want: 0},
		{name: "second page", page: 2, limit: 20, want: 20},
		{name: "page zero treated as first", page: 0, limit: 20, want: 0},
		{name: "negative page treated as first", page: -3, limit: 20, want: 0},
		{name: "offset beyond int32", page: math.MaxInt32, limit: 100, wantErr: errOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := pageOffset(tt.page, tt.limit)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("pageOffset(%d, %d) error = %v, want %v", tt.page, tt.limit, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("pageOffset(%d, %d) = %d, want %d", tt.page, tt.limit, got, tt.want)
			}
		})
	}
}

func TestToInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      int
		want    int32
		wantErr error
	}{
		{name: "zero", in: 0, want: 0},
		{name: "max", in: math.MaxInt32, want: math.MaxInt32},
		{name: "min", in: math.MinInt32, want: math.MinInt32},
		{name: "above max", in: math.MaxInt32 + 1, wantErr: errOutOfRange},
		{name: "below min", in: math.MinInt32 - 1, wantErr: errOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := toInt32(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("toInt32(%d) error = %v, want %v", tt.in, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("toInt32(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestMapInsertTransactionError(t *testing.T) {
	t.Parallel()

	other := errors.New("connection reset")

	tests := []struct {
		name string
		in   error
		want error
	}{
		{
			name: "one-reversal index means already reversed",
			in:   &pgconn.PgError{Code: pgErrUniqueViolation, ConstraintName: uniqueOneReversal},
			want: transaction.ErrAlreadyReversed,
		},
		{
			name: "reference constraint means duplicate reference",
			in:   &pgconn.PgError{Code: pgErrUniqueViolation, ConstraintName: "transactions_reference_key"},
			want: transaction.ErrDuplicateReference,
		},
		{
			name: "index name on a non-unique error is not a reversal conflict",
			in:   &pgconn.PgError{Code: "23503", ConstraintName: uniqueOneReversal},
			want: nil,
		},
		{name: "other errors pass through", in: other, want: other},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mapInsertTransactionError(tt.in)

			want := tt.want
			if want == nil {
				want = tt.in
			}

			if !errors.Is(got, want) {
				t.Errorf("mapInsertTransactionError(%v) = %v, want %v", tt.in, got, want)
			}
		})
	}
}

func TestMapIntegrityError(t *testing.T) {
	t.Parallel()

	other := errors.New("connection reset")
	unique := &pgconn.PgError{Code: pgErrUniqueViolation}

	tests := []struct {
		name string
		in   error
		want error
	}{
		{name: "currency mismatch", in: &pgconn.PgError{Code: pgErrCurrencyMismatch}, want: transaction.ErrCurrencyMismatch},
		{
			name: "wrapped unbalanced",
			in:   fmt.Errorf("copy: %w", &pgconn.PgError{Code: pgErrUnbalanced}),
			want: transaction.ErrUnbalancedTransaction,
		},
		{name: "other postgres error passes through", in: unique, want: unique},
		{name: "non-postgres error passes through", in: other, want: other},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapIntegrityError(tt.in); !errors.Is(got, tt.want) {
				t.Errorf("mapIntegrityError(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestMapToAccountDomainList_EmptyIsNotNil(t *testing.T) {
	t.Parallel()

	if got := mapToAccountDomainList(nil); got == nil || len(got) != 0 {
		t.Errorf("mapToAccountDomainList(nil) = %#v, want an empty non-nil slice (JSON [])", got)
	}
}
