package transaction

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// fakeRepo implements Repository; unexercised methods panic via the nil embed.
type fakeRepo struct {
	Repository

	created  []CreateTransactionParams
	listed   []ListTransactionsParams
	listRows int
	fetched  []uuid.UUID
	reversed []ReverseTransactionParams
}

func (f *fakeRepo) Reverse(_ context.Context, p ReverseTransactionParams) (*Transaction, error) {
	f.reversed = append(f.reversed, p)
	return &Transaction{Reference: p.Reference, Type: TypeReversal}, nil
}

func (f *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (*Transaction, error) {
	f.fetched = append(f.fetched, id)
	return &Transaction{ID: id}, nil
}

func (f *fakeRepo) List(_ context.Context, p ListTransactionsParams) ([]Transaction, error) {
	f.listed = append(f.listed, p)

	rows := make([]Transaction, f.listRows)
	for i := range rows {
		rows[i].ID = uuid.New()
	}

	return rows, nil
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

func (fakeRegistry) List() []currency.Currency { return nil }

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
		{
			name: "unbalanced",
			params: CreateTransactionParams{
				Reference: "r2",
				Currency:  currency.NGN,
				Entries:   []EntryParams{debit(1), credit(2)},
			},
			wantErr: ErrUnbalancedTransaction,
		},
		{
			name:    "reversal type is reserved",
			params:  CreateTransactionParams{Reference: "r4", Type: TypeReversal, Currency: currency.NGN, Entries: balanced},
			wantErr: ErrReservedType,
		},
		{
			name:    "inactive currency",
			params:  CreateTransactionParams{Reference: "r3", Currency: currency.USD, Entries: balanced},
			wantErr: ErrInvalidCurrency,
		},
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

			if saved := string(repo.created[0].Metadata); saved != "{}" {
				t.Errorf("stored metadata = %s, want {} when omitted", saved)
			}
		})
	}
}

func TestTransactionService_GetTransactionDetails(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	id := uuid.New()

	got, err := newTestService(repo).GetTransactionDetails(t.Context(), id)
	if err != nil {
		t.Fatalf("GetTransactionDetails() error = %v", err)
	}

	if got.ID != id || len(repo.fetched) != 1 || repo.fetched[0] != id {
		t.Errorf("GetTransactionDetails() = %+v, fetched %v, want id %s", got, repo.fetched, id)
	}
}

func TestTransactionService_ListTransactions(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		params    ListTransactionsParams
		wantLimit int32
		wantErr   error
	}{
		{name: "omitted limit uses default", wantLimit: DefaultPageLimit},
		{name: "negative limit uses default", params: ListTransactionsParams{Limit: -1}, wantLimit: DefaultPageLimit},
		{name: "limit within bounds is kept", params: ListTransactionsParams{Limit: 7}, wantLimit: 7},
		{name: "limit is capped", params: ListTransactionsParams{Limit: 500}, wantLimit: MaxPageLimit},
		{
			name:      "start before end",
			params:    ListTransactionsParams{StartDate: now, EndDate: now.Add(time.Hour)},
			wantLimit: DefaultPageLimit,
		},
		{name: "only start date", params: ListTransactionsParams{StartDate: now}, wantLimit: DefaultPageLimit},
		{name: "only end date", params: ListTransactionsParams{EndDate: now}, wantLimit: DefaultPageLimit},
		{
			name:    "start after end",
			params:  ListTransactionsParams{StartDate: now.Add(time.Hour), EndDate: now},
			wantErr: ErrInvalidDateRange,
		},
		{
			name:    "start equal to end",
			params:  ListTransactionsParams{StartDate: now, EndDate: now},
			wantErr: ErrInvalidDateRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}

			_, err := newTestService(repo).ListTransactions(t.Context(), tt.params)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ListTransactions() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if len(repo.listed) != 0 {
					t.Error("rejected listing was passed to the repository")
				}

				return
			}

			// One extra row is fetched to tell whether there's another page.
			if len(repo.listed) != 1 || repo.listed[0].Limit != tt.wantLimit+1 {
				t.Errorf("repository got %+v, want limit %d", repo.listed, tt.wantLimit+1)
			}
		})
	}
}

func TestTransactionService_ReverseTransaction(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	params := ReverseTransactionParams{ID: uuid.New(), Reference: "rev-1", Reason: "duplicate charge"}

	got, err := newTestService(repo).ReverseTransaction(t.Context(), params)
	if err != nil {
		t.Fatalf("ReverseTransaction() error = %v", err)
	}

	if len(repo.reversed) != 1 || repo.reversed[0] != params {
		t.Errorf("repository got %+v, want %+v", repo.reversed, params)
	}

	if got.Type != TypeReversal {
		t.Errorf("ReverseTransaction() type = %q, want %q", got.Type, TypeReversal)
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
		{name: "CreateConversion", call: func() error {
			_, err := svc.CreateConversion(t.Context(), CreateConversionParams{})
			return err
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

func TestTransactionService_ListTransactions_Page(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{listRows: 3}

	page, err := newTestService(repo).ListTransactions(t.Context(), ListTransactionsParams{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 2 || !page.HasMore || page.NextCursor == nil || *page.NextCursor != page.Items[1].ID {
		t.Errorf("page = %+v, want 2 items, has_more and next_cursor = the second item's id", page)
	}
}
