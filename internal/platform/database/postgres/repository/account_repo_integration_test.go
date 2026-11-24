//go:build integration

package repository_test

import (
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/repository"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
)

// TestBalanceSignConvention checks that balance, snapshot, reconciliation and
// statement queries agree on the normal-balance sign rule for both a
// credit-normal (USER_WALLET) and a debit-normal (PLATFORM_CASH) account.
//
// It needs a migrated and seeded database in TEST_DATABASE_URL. Entries are
// immutable, so it creates fresh owners and leaves its rows behind.
func TestBalanceSignConvention(t *testing.T) {
	t.Parallel()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := t.Context()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	queries := sqlc.New(pool)
	accounts := repository.NewAccountRepository(pool, queries)
	transactions := repository.NewTransactionRepository(pool, queries)

	owner := "it-" + uuid.NewString()

	wallet := mustCreateAccount(t, accounts, owner, account.TypeUserWallet)
	cash := mustCreateAccount(t, accounts, owner, account.TypePlatformCash)

	windowStart := time.Now().Add(-time.Hour)

	// Fund the wallet: cash (asset) goes up by debit, wallet (liability) by credit.
	mustPost(t, transactions, cash.ID, wallet.ID, 10_000)

	assertBalance(t, accounts, wallet.ID, 10_000)
	assertBalance(t, accounts, cash.ID, 10_000)

	snapshot, err := accounts.CreateBalanceSnapshot(ctx, wallet.ID)
	if err != nil {
		t.Fatalf("CreateBalanceSnapshot: %v", err)
	}

	if snapshot.Balance != 10_000 {
		t.Errorf("snapshot balance = %d, want 10000", snapshot.Balance)
	}

	assertReconciled(t, accounts, wallet.ID)

	// Spend from the wallet after the snapshot to exercise snapshot + delta.
	mustPost(t, transactions, wallet.ID, cash.ID, 2_500)

	assertBalance(t, accounts, wallet.ID, 7_500)
	assertBalance(t, accounts, cash.ID, 7_500)
	assertReconciled(t, accounts, wallet.ID)

	assertStatement(t, accounts, wallet.ID, windowStart, time.Now().Add(time.Hour))
}

func mustCreateAccount(t *testing.T, repo account.Repository, owner string, accountType account.Type) *account.Account {
	t.Helper()

	acc, err := repo.Create(t.Context(), account.CreateAccountParams{
		OwnerID:  owner,
		Type:     string(accountType),
		Currency: currency.NGN.String(),
	})
	if err != nil {
		t.Fatalf("create %s account: %v", accountType, err)
	}

	return acc
}

// mustPost records a balanced transfer: debit one account, credit the other.
func mustPost(t *testing.T, repo transaction.Repository, debit, credit uuid.UUID, amount int64) {
	t.Helper()

	_, err := repo.Create(t.Context(), transaction.CreateTransactionParams{
		Reference:       "it-" + uuid.NewString(),
		Type:            itTxnType,
		Currency:        currency.NGN,
		Metadata:        []byte("{}"),
		TransactionDate: time.Now(),
		Entries: []transaction.EntryParams{
			{AccountID: debit, Type: domain.EntryTypeDebit, Amount: amount},
			{AccountID: credit, Type: domain.EntryTypeCredit, Amount: amount},
		},
	})
	if err != nil {
		t.Fatalf("post transaction: %v", err)
	}
}

func assertBalance(t *testing.T, repo account.Repository, id uuid.UUID, want int64) {
	t.Helper()

	got, err := repo.GetBalance(t.Context(), id)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}

	if got.BalanceMinor != want {
		t.Errorf("balance of %s = %d, want %d", id, got.BalanceMinor, want)
	}
}

func assertReconciled(t *testing.T, repo account.Repository, id uuid.UUID) {
	t.Helper()

	result, err := repo.Reconcile(t.Context(), id)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if result.Status != account.ReconciliationStatusReconciled || result.DiscrepancyAmount != 0 {
		t.Errorf("reconcile = %s with discrepancy %d (ledger %d, snapshot %d + delta %d); want RECONCILED",
			result.Status, result.DiscrepancyAmount,
			result.RawLedgerSum, result.SnapshotBalance, result.DeltaSinceSnapshot)
	}
}

// assertStatement checks the wallet's statement after a 10000 credit then a
// 2500 debit: totals, closing balance and per-entry running balances.
func assertStatement(t *testing.T, repo account.Repository, id uuid.UUID, start, end time.Time) {
	t.Helper()

	summary, err := repo.GetStatementSummary(t.Context(), account.GetStatementSummaryParams{
		AccountID: id,
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		t.Fatalf("GetStatementSummary: %v", err)
	}

	if summary.OpeningBalance != 0 || summary.TotalCredits != 10_000 ||
		summary.TotalDebits != 2_500 || summary.ClosingBalance != 7_500 {
		t.Errorf("summary = opening %d, credits %d, debits %d, closing %d; want 0, 10000, 2500, 7500",
			summary.OpeningBalance, summary.TotalCredits, summary.TotalDebits, summary.ClosingBalance)
	}

	entries, err := repo.ListStatementEntries(t.Context(), account.GetStatementEntriesParams{
		AccountID: id,
		StartDate: start,
		EndDate:   end,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListStatementEntries: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("statement entries = %d, want 2", len(entries))
	}

	if entries[0].RunningBalance != 7_500 || entries[1].RunningBalance != 10_000 {
		t.Errorf("running balances (newest first) = %d, %d; want 7500, 10000",
			entries[0].RunningBalance, entries[1].RunningBalance)
	}
}

// newAccountRepo connects to TEST_DATABASE_URL, or skips the test when it is unset.
func newAccountRepo(t *testing.T) account.Repository {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	return repository.NewAccountRepository(pool, sqlc.New(pool))
}

func TestChangeAccountStatus(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	accounts := newAccountRepo(t)
	wallet := mustCreateAccount(t, accounts, "it-"+uuid.NewString(), account.TypeUserWallet)

	active := account.StatusActive
	freeze := account.StatusChangeParams{
		AccountID:      wallet.ID,
		NewStatus:      account.StatusFullyFrozen,
		ExpectedStatus: &active,
		Reason:         "kyc",
		ActorType:      account.ActorTypeAdmin,
		ActorID:        "admin-1",
	}

	history, err := accounts.ChangeAccountStatus(ctx, freeze)
	if err != nil {
		t.Fatalf("ChangeAccountStatus: %v", err)
	}

	if history.PreviousStatus == nil || *history.PreviousStatus != account.StatusActive ||
		history.NewStatus != account.StatusFullyFrozen {
		t.Errorf("history = %+v, want ACTIVE -> FULLY_FROZEN", history)
	}

	// The account is FULLY_FROZEN now, so expecting ACTIVE must fail.
	if _, err := accounts.ChangeAccountStatus(ctx, freeze); !errors.Is(err, account.ErrStatusConflict) {
		t.Errorf("repeat ChangeAccountStatus error = %v, want ErrStatusConflict", err)
	}

	got, err := accounts.ListAccountStatusHistory(ctx, account.ListAccountStatusHistoryParams{
		AccountID: wallet.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListAccountStatusHistory: %v", err)
	}

	if len(got) != 1 || got[0].ID != history.ID {
		t.Errorf("history list = %+v, want only %s", got, history.ID)
	}
}

func TestResolveDiscrepancy(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	accounts := newAccountRepo(t)
	wallet := mustCreateAccount(t, accounts, "it-"+uuid.NewString(), account.TypeUserWallet)

	record, err := accounts.CreateDiscrepancyRecord(ctx, account.CreateDiscrepancyRecordParams{
		AccountID:         wallet.ID,
		DiscrepancyAmount: 1,
	})
	if err != nil {
		t.Fatalf("CreateDiscrepancyRecord: %v", err)
	}

	if err := accounts.ResolveDiscrepancy(ctx, record.ID, "test"); err != nil {
		t.Fatalf("ResolveDiscrepancy: %v", err)
	}

	if err := accounts.ResolveDiscrepancy(ctx, record.ID, "again"); !errors.Is(err, account.ErrDiscrepancyNotFound) {
		t.Errorf("second ResolveDiscrepancy error = %v, want ErrDiscrepancyNotFound", err)
	}

	if err := accounts.ResolveDiscrepancy(ctx, uuid.New(), "x"); !errors.Is(err, account.ErrDiscrepancyNotFound) {
		t.Errorf("unknown ResolveDiscrepancy error = %v, want ErrDiscrepancyNotFound", err)
	}
}

// TestChangeAccountStatus_Transitions checks that a no-op change and any
// change out of CLOSED are rejected without writing history.
func TestChangeAccountStatus_Transitions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	accounts := newAccountRepo(t)
	wallet := mustCreateAccount(t, accounts, "it-"+uuid.NewString(), account.TypeUserWallet)

	change := func(status account.Status) error {
		_, err := accounts.ChangeAccountStatus(ctx, account.StatusChangeParams{
			AccountID: wallet.ID, NewStatus: status, Reason: "it", ActorType: account.ActorTypeAdmin, ActorID: "it",
		})

		return err
	}

	if err := change(account.StatusActive); !errors.Is(err, account.ErrStatusUnchanged) {
		t.Errorf("ACTIVE -> ACTIVE error = %v, want ErrStatusUnchanged", err)
	}

	if err := change(account.StatusClosed); err != nil {
		t.Fatalf("ACTIVE -> CLOSED: %v", err)
	}

	if err := change(account.StatusActive); !errors.Is(err, account.ErrAccountClosed) {
		t.Errorf("CLOSED -> ACTIVE error = %v, want ErrAccountClosed", err)
	}

	history, err := accounts.ListAccountStatusHistory(ctx, account.ListAccountStatusHistoryParams{
		AccountID: wallet.ID, Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListAccountStatusHistory: %v", err)
	}

	if len(history) != 1 || history[0].NewStatus != account.StatusClosed {
		t.Errorf("history = %+v, want only the ACTIVE -> CLOSED change", history)
	}
}

// TestListAccounts_OwnerFilter creates accounts for two owners and checks that
// filtering by owner returns exactly that owner's accounts, alone or combined
// with the type filter.
func TestListAccounts_OwnerFilter(t *testing.T) {
	t.Parallel()

	accounts := newAccountRepo(t)

	alice, bob := "it-"+uuid.NewString(), "it-"+uuid.NewString()
	aliceWallet := mustCreateAccount(t, accounts, alice, account.TypeUserWallet)
	aliceCash := mustCreateAccount(t, accounts, alice, account.TypePlatformCash)
	mustCreateAccount(t, accounts, bob, account.TypeUserWallet)

	tests := []struct {
		name    string
		params  account.ListAccountsParams
		wantIDs []uuid.UUID
		ordered bool // wantIDs is the exact order, not just the set
	}{
		{
			name:    "owner only",
			params:  account.ListAccountsParams{OwnerID: alice},
			wantIDs: []uuid.UUID{aliceWallet.ID, aliceCash.ID},
		},
		{
			name:    "owner and type",
			params:  account.ListAccountsParams{OwnerID: alice, Type: string(account.TypeUserWallet)},
			wantIDs: []uuid.UUID{aliceWallet.ID},
		},
		{
			name:    "sorted by type ascending",
			params:  account.ListAccountsParams{OwnerID: alice, SortBy: account.SortByType, OrderBy: account.SortOrderAsc},
			wantIDs: []uuid.UUID{aliceCash.ID, aliceWallet.ID}, // PLATFORM_CASH < USER_WALLET
			ordered: true,
		},
		{
			name:   "unknown owner",
			params: account.ListAccountsParams{OwnerID: "it-" + uuid.NewString()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.params.Page, tt.params.Limit = 1, 100

			got, err := accounts.List(t.Context(), tt.params)
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			gotIDs := make([]uuid.UUID, len(got))
			for i, a := range got {
				gotIDs[i] = a.ID

				if a.OwnerID != tt.params.OwnerID {
					t.Errorf("account %s has owner %q, want %q", a.ID, a.OwnerID, tt.params.OwnerID)
				}
			}

			if tt.ordered && !slices.Equal(gotIDs, tt.wantIDs) {
				t.Errorf("List ids = %v, want exactly %v", gotIDs, tt.wantIDs)
			}

			if !sameIDs(gotIDs, tt.wantIDs) {
				t.Errorf("List ids = %v, want %v (any order)", gotIDs, tt.wantIDs)
			}

			count, err := accounts.Count(t.Context(), tt.params)
			if err != nil || count != int64(len(tt.wantIDs)) {
				t.Errorf("Count = %d, %v; want %d", count, err, len(tt.wantIDs))
			}
		})
	}
}

// sameIDs reports whether a and b hold the same ids, ignoring order.
func sameIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}

	seen := make(map[uuid.UUID]int, len(a))
	for _, id := range a {
		seen[id]++
	}

	for _, id := range b {
		if seen[id] == 0 {
			return false
		}

		seen[id]--
	}

	return true
}
