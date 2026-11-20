package transaction

import (
	"context"
	"errors"
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// fakeRepo implements Repository; unexercised methods panic via the nil embed.
type fakeRepo struct {
	Repository

	created []CreateTransactionParams
}

func (f *fakeRepo) Create(_ context.Context, p CreateTransactionParams) (*Transaction, error) {
	f.created = append(f.created, p)
	return &Transaction{Reference: p.Reference}, nil
}

type fakeRegistry struct{ active map[currency.Code]bool }

func (r fakeRegistry) Get(code currency.Code) (currency.Currency, bool) {
	return currency.Currency{Code: code}, r.active[code]
}

func (fakeRegistry) Reload(context.Context) error { return nil }

func newTestService(repo *fakeRepo) Service {
	return NewTransactionService(repo, fakeRegistry{active: map[currency.Code]bool{currency.NGN: true}})
}

func TestTransactionService_CreateTransaction(t *testing.T) {
	t.Parallel()

	balanced := []EntryParams{debit(100), credit(100)}

	tests := []struct {
		name    string
		params  CreateTransactionParams
		wantErr error
	}{
		{name: "valid", params: CreateTransactionParams{Reference: "r1", Currency: currency.NGN, Entries: balanced}},
		{name: "unbalanced", params: CreateTransactionParams{Reference: "r2", Currency: currency.NGN, Entries: []EntryParams{debit(1), credit(2)}}, wantErr: ErrUnbalancedTransaction},
		{name: "inactive currency", params: CreateTransactionParams{Reference: "r3", Currency: currency.USD, Entries: balanced}, wantErr: ErrInvalidCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}

			got, err := newTestService(repo).CreateTransaction(t.Context(), tt.params)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateTransaction() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if len(repo.created) != 0 {
					t.Error("rejected transaction was passed to the repository")
				}

				return
			}

			if got == nil || got.Reference != tt.params.Reference {
				t.Errorf("CreateTransaction() = %+v, want reference %q", got, tt.params.Reference)
			}
		})
	}
}

// TestTransactionService_Unimplemented pins the 501 contract for operations
// with no persistence yet, so they cannot silently report success again.
func TestTransactionService_Unimplemented(t *testing.T) {
	t.Parallel()

	svc := newTestService(&fakeRepo{})

	tests := []struct {
		name string
		call func() error
	}{
		{name: "GetTransactionDetails", call: func() error {
			_, err := svc.GetTransactionDetails(t.Context(), "id")
			return err
		}},
		{name: "ListTransactions", call: func() error {
			_, err := svc.ListTransactions(t.Context(), ListTransactionsParams{})
			return err
		}},
		{name: "ReverseTransaction", call: func() error {
			return svc.ReverseTransaction(t.Context(), "id")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.call(); !errors.Is(err, domain.ErrNotImplemented) {
				t.Errorf("error = %v, want domain.ErrNotImplemented", err)
			}
		})
	}
}
