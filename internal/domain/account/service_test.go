package account

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// fakeRepo implements Repository. Methods a test doesn't override panic via the
// nil embedded interface, which flags unexpected calls.
type fakeRepo struct {
	Repository

	createCalls      int
	listParams       ListAccountsParams
	entriesParams    GetStatementEntriesParams
	reconcileResult  *ReconciliationResult
	discrepancies    []CreateDiscrepancyRecordParams
	discrepancyErr   error
	getByIDErr       error
	snapshotAccounts []uuid.UUID
}

func (f *fakeRepo) Create(_ context.Context, p CreateAccountParams) (*Account, error) {
	f.createCalls++
	return &Account{OwnerID: p.OwnerID}, nil
}

func (f *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (*Account, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}

	return &Account{ID: id}, nil
}

func (f *fakeRepo) List(_ context.Context, p ListAccountsParams) ([]Account, error) {
	f.listParams = p
	return nil, nil
}

func (f *fakeRepo) GetStatementSummary(_ context.Context, p GetStatementSummaryParams) (*StatementSummary, error) {
	return &StatementSummary{AccountID: p.AccountID}, nil
}

func (f *fakeRepo) ListStatementEntries(_ context.Context, p GetStatementEntriesParams) ([]StatementEntry, error) {
	f.entriesParams = p
	return nil, nil
}

func (f *fakeRepo) Reconcile(_ context.Context, _ uuid.UUID) (*ReconciliationResult, error) {
	return f.reconcileResult, nil
}

func (f *fakeRepo) CreateDiscrepancyRecord(_ context.Context, p CreateDiscrepancyRecordParams) (*DiscrepancyRecord, error) {
	if f.discrepancyErr != nil {
		return nil, f.discrepancyErr
	}

	f.discrepancies = append(f.discrepancies, p)

	return &DiscrepancyRecord{}, nil
}

func (f *fakeRepo) CreateBalanceSnapshot(_ context.Context, id uuid.UUID) (*BalanceSnapshot, error) {
	f.snapshotAccounts = append(f.snapshotAccounts, id)
	return &BalanceSnapshot{AccountID: id}, nil
}

type fakeRegistry struct{ active map[currency.Code]bool }

func (r fakeRegistry) Get(code currency.Code) (currency.Currency, bool) {
	return currency.Currency{Code: code}, r.active[code]
}

func (fakeRegistry) Reload(context.Context) error { return nil }

func newTestService(repo *fakeRepo) Service {
	return NewAccountService(repo, fakeRegistry{active: map[currency.Code]bool{currency.NGN: true}}, slog.New(slog.DiscardHandler))
}

func TestAccountService_CreateAccount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  CreateAccountParams
		wantErr error
	}{
		{name: "valid", params: CreateAccountParams{OwnerID: "u1", Type: "USER_WALLET", Currency: "NGN"}},
		{name: "unknown type", params: CreateAccountParams{OwnerID: "u1", Type: "NOPE", Currency: "NGN"}, wantErr: ErrInvalidType},
		{name: "inactive currency", params: CreateAccountParams{OwnerID: "u1", Type: "USER_WALLET", Currency: "USD"}, wantErr: ErrInvalidCurrency},
		{name: "unknown currency", params: CreateAccountParams{OwnerID: "u1", Type: "USER_WALLET", Currency: "XXX"}, wantErr: ErrInvalidCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}

			got, err := newTestService(repo).CreateAccount(t.Context(), tt.params)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateAccount() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if repo.createCalls != 0 {
					t.Error("invalid account was passed to the repository")
				}

				return
			}

			if got == nil || got.OwnerID != tt.params.OwnerID {
				t.Errorf("CreateAccount() = %+v, want account for owner %q", got, tt.params.OwnerID)
			}
		})
	}
}

func TestAccountService_ListAccounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		page, limit         int
		wantPage, wantLimit int
	}{
		{name: "omitted page and limit use defaults", page: 0, limit: 0, wantPage: 1, wantLimit: DefaultPageLimit},
		{name: "negative values use defaults", page: -2, limit: -5, wantPage: 1, wantLimit: DefaultPageLimit},
		{name: "explicit values pass through", page: 3, limit: 50, wantPage: 3, wantLimit: 50},
		{name: "limit is capped", page: 1, limit: 500, wantPage: 1, wantLimit: MaxPageLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}
			if _, err := newTestService(repo).ListAccounts(t.Context(), ListAccountsParams{Page: tt.page, Limit: tt.limit}); err != nil {
				t.Fatalf("ListAccounts() error = %v", err)
			}

			if repo.listParams.Page != tt.wantPage || repo.listParams.Limit != tt.wantLimit {
				t.Errorf("queried page=%d limit=%d, want page=%d limit=%d",
					repo.listParams.Page, repo.listParams.Limit, tt.wantPage, tt.wantLimit)
			}
		})
	}
}

func TestAccountService_GetAccountStatementSummary(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name       string
		start, end time.Time
		wantErr    error
	}{
		{name: "start before end", start: now.Add(-time.Hour), end: now},
		{name: "single instant", start: now, end: now},
		{name: "start after end", start: now, end: now.Add(-time.Hour), wantErr: ErrInvalidDateRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := newTestService(&fakeRepo{}).GetAccountStatementSummary(t.Context(),
				GetStatementSummaryParams{StartDate: tt.start, EndDate: tt.end})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetAccountStatementSummary() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccountService_GetAccountStatementEntries(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name       string
		start, end time.Time
		limit      int32
		wantLimit  int32
		wantErr    error
	}{
		{name: "omitted limit uses default", start: now, end: now, limit: 0, wantLimit: DefaultPageLimit},
		{name: "explicit limit passes through", start: now, end: now, limit: 7, wantLimit: 7},
		{name: "limit is capped", start: now, end: now, limit: 1000, wantLimit: MaxPageLimit},
		{name: "start after end", start: now, end: now.Add(-time.Hour), wantErr: ErrInvalidDateRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}

			_, err := newTestService(repo).GetAccountStatementEntries(t.Context(),
				GetStatementEntriesParams{StartDate: tt.start, EndDate: tt.end, Limit: tt.limit})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetAccountStatementEntries() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && repo.entriesParams.Limit != tt.wantLimit {
				t.Errorf("queried limit = %d, want %d", repo.entriesParams.Limit, tt.wantLimit)
			}
		})
	}
}

func TestAccountService_ReconcileAccount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		status      string
		wantRecords int
	}{
		{name: "reconciled records nothing", status: ReconciliationStatusReconciled, wantRecords: 0},
		{name: "discrepancy is recorded", status: ReconciliationStatusDiscrepancy, wantRecords: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{reconcileResult: &ReconciliationResult{Status: tt.status, DiscrepancyAmount: 42}}

			result, err := newTestService(repo).ReconcileAccount(t.Context(), uuid.New())
			if err != nil {
				t.Fatalf("ReconcileAccount() error = %v", err)
			}

			if result.Status != tt.status {
				t.Errorf("status = %q, want %q", result.Status, tt.status)
			}

			if len(repo.discrepancies) != tt.wantRecords {
				t.Fatalf("discrepancy records = %d, want %d", len(repo.discrepancies), tt.wantRecords)
			}

			if tt.wantRecords > 0 && repo.discrepancies[0].DiscrepancyAmount != 42 {
				t.Errorf("recorded amount = %d, want 42", repo.discrepancies[0].DiscrepancyAmount)
			}
		})
	}
}

func TestAccountService_ReconcileAccount_RecordFailure(t *testing.T) {
	t.Parallel()

	errStore := errors.New("insert failed")
	repo := &fakeRepo{
		reconcileResult: &ReconciliationResult{Status: ReconciliationStatusDiscrepancy},
		discrepancyErr:  errStore,
	}

	// A discrepancy that cannot be persisted must not be reported as a normal result.
	if _, err := newTestService(repo).ReconcileAccount(t.Context(), uuid.New()); !errors.Is(err, errStore) {
		t.Fatalf("ReconcileAccount() error = %v, want wrapped %v", err, errStore)
	}
}

func TestAccountService_CreateAccountSnapshot(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	tests := []struct {
		name          string
		getByIDErr    error
		wantErr       error
		wantSnapshots int
	}{
		{name: "existing account", wantSnapshots: 1},
		{name: "missing account", getByIDErr: ErrAccountNotFound, wantErr: ErrAccountNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{getByIDErr: tt.getByIDErr}

			snapshot, err := newTestService(repo).CreateAccountSnapshot(t.Context(), CreateAccountSnapshotParams{ID: id, Reason: "eod"})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateAccountSnapshot() error = %v, want %v", err, tt.wantErr)
			}

			if len(repo.snapshotAccounts) != tt.wantSnapshots {
				t.Fatalf("snapshots created = %d, want %d", len(repo.snapshotAccounts), tt.wantSnapshots)
			}

			if tt.wantErr == nil && snapshot.AccountID != id {
				t.Errorf("snapshot account = %s, want %s", snapshot.AccountID, id)
			}
		})
	}
}
