//go:build integration

package repository_test

import (
	"errors"
	"os"
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

// itReason is the reason given for reversals posted by these tests.
const itReason = "integration test"

// itTxnType is the type of transactions posted by these tests.
const itTxnType = "INTEGRATION_TEST"

// readBackFixture is two transactions between a fresh wallet and cash account:
// first funds the wallet, second spends from it.
type readBackFixture struct {
	repo          transaction.Repository
	walletID      uuid.UUID
	first, second *transaction.Transaction
}

// TestTransactionReadBack posts two transactions between fresh accounts and
// reads them back by id and through the filtered, cursor-paged listing.
func TestTransactionReadBack(t *testing.T) {
	t.Parallel()

	accounts, transactions := newRepos(t)

	owner := "it-" + uuid.NewString()
	wallet := mustCreateAccount(t, accounts, owner, account.TypeUserWallet)
	cash := mustCreateAccount(t, accounts, owner, account.TypePlatformCash)

	f := readBackFixture{
		repo:     transactions,
		walletID: wallet.ID,
		first:    mustPostReturning(t, transactions, cash.ID, wallet.ID, 1_000),
		second:   mustPostReturning(t, transactions, wallet.ID, cash.ID, 400),
	}

	if len(f.first.Entries) != 2 || f.first.Entries[0].ID == uuid.Nil {
		t.Fatalf("Create returned entries %+v, want 2 persisted entries", f.first.Entries)
	}

	t.Run("get by id includes entries", func(t *testing.T) { t.Parallel(); checkGetByID(t, f) })
	t.Run("unknown id is not found", func(t *testing.T) { t.Parallel(); checkGetByIDNotFound(t, f) })
	t.Run("account filter pages newest first", func(t *testing.T) { t.Parallel(); checkListPaging(t, f) })
	t.Run("reference and date filters", func(t *testing.T) { t.Parallel(); checkListFilters(t, f) })
}

func checkGetByID(t *testing.T, f readBackFixture) {
	t.Helper()

	got, err := f.repo.GetByID(t.Context(), f.second.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Reference != f.second.Reference || len(got.Entries) != 2 {
		t.Fatalf("GetByID = %+v, want reference %q with 2 entries", got, f.second.Reference)
	}

	if got.Entries[0].EntrySequence >= got.Entries[1].EntrySequence {
		t.Errorf("entries not in sequence order: %+v", got.Entries)
	}
}

func checkGetByIDNotFound(t *testing.T, f readBackFixture) {
	t.Helper()

	if _, err := f.repo.GetByID(t.Context(), uuid.New()); !errors.Is(err, transaction.ErrTransactionNotFound) {
		t.Errorf("GetByID(unknown) error = %v, want ErrTransactionNotFound", err)
	}
}

func checkListPaging(t *testing.T, f readBackFixture) {
	t.Helper()

	filter := transaction.ListTransactionsParams{AccountID: uuid.NullUUID{UUID: f.walletID, Valid: true}, Limit: 1}

	// Newest first: second, then first, then an empty page.
	for page, want := range []*transaction.Transaction{f.second, f.first, nil} {
		got, err := f.repo.List(t.Context(), filter)
		if err != nil {
			t.Fatalf("page %d: %v", page+1, err)
		}

		if want == nil {
			if len(got) != 0 {
				t.Fatalf("page %d = %+v, want empty", page+1, got)
			}

			return
		}

		if len(got) != 1 || got[0].ID != want.ID {
			t.Fatalf("page %d = %+v, want only %s", page+1, got, want.ID)
		}

		filter.Cursor = uuid.NullUUID{UUID: got[0].ID, Valid: true}
	}
}

func checkListFilters(t *testing.T, f readBackFixture) {
	t.Helper()

	got, err := f.repo.List(t.Context(), transaction.ListTransactionsParams{
		Reference: f.first.Reference,
		Currency:  currency.NGN,
		StartDate: f.first.CreatedAt.Add(-time.Minute),
		EndDate:   f.first.CreatedAt.Add(time.Minute),
		Limit:     10,
	})
	if err != nil || len(got) != 1 || got[0].ID != f.first.ID {
		t.Fatalf("List(reference) = %+v, %v; want only %s", got, err, f.first.ID)
	}

	got, err = f.repo.List(t.Context(), transaction.ListTransactionsParams{
		Reference: f.first.Reference,
		EndDate:   f.first.CreatedAt,
		Limit:     10,
	})
	if err != nil || len(got) != 0 {
		t.Fatalf("List(end_date = created_at) = %+v, %v; want empty (end is exclusive)", got, err)
	}
}

// newRepos connects to TEST_DATABASE_URL, or skips the test when it is unset.
func newRepos(t *testing.T) (account.Repository, transaction.Repository) {
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

	queries := sqlc.New(pool)

	return repository.NewAccountRepository(pool, queries), repository.NewTransactionRepository(pool, queries)
}

// mustPostReturning is mustPost, returning the created transaction.
func mustPostReturning(
	t *testing.T,
	repo transaction.Repository,
	debit, credit uuid.UUID,
	amount int64,
) *transaction.Transaction {
	t.Helper()

	created, err := repo.Create(t.Context(), transaction.CreateTransactionParams{
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

	return created
}

// TestTransactionReversal reverses a funding transaction and checks the mirror
// entries, the restored balances and every rejection path.
func TestTransactionReversal(t *testing.T) {
	t.Parallel()

	accounts, transactions := newRepos(t)

	owner := "it-" + uuid.NewString()
	wallet := mustCreateAccount(t, accounts, owner, account.TypeUserWallet)
	cash := mustCreateAccount(t, accounts, owner, account.TypePlatformCash)

	original := mustPostReturning(t, transactions, cash.ID, wallet.ID, 1_000)
	reversal := mustReverse(t, transactions, original.ID)

	checkMirror(t, original, reversal)
	assertBalance(t, accounts, wallet.ID, 0)
	assertBalance(t, accounts, cash.ID, 0)

	tests := []struct {
		name      string
		id        uuid.UUID
		reference string
		want      error
	}{
		{name: "second reversal", id: original.ID, want: transaction.ErrAlreadyReversed},
		{name: "reversing a reversal", id: reversal.ID, want: transaction.ErrCannotReverseReversal},
		{name: "unknown transaction", id: uuid.New(), want: transaction.ErrTransactionNotFound},
		{
			name:      "reused reference",
			id:        mustPostReturning(t, transactions, cash.ID, wallet.ID, 5).ID,
			reference: reversal.Reference,
			want:      transaction.ErrDuplicateReference,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reference := tt.reference
			if reference == "" {
				reference = "it-rev-" + uuid.NewString()
			}

			_, err := transactions.Reverse(t.Context(), transaction.ReverseTransactionParams{
				ID: tt.id, Reference: reference, Reason: itReason,
			})
			if !errors.Is(err, tt.want) {
				t.Errorf("Reverse() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestTransactionReversal_Concurrent races two reversals of one transaction:
// exactly one wins, and the unique index turns the other into ErrAlreadyReversed.
func TestTransactionReversal_Concurrent(t *testing.T) {
	t.Parallel()

	accounts, transactions := newRepos(t)

	owner := "it-" + uuid.NewString()
	wallet := mustCreateAccount(t, accounts, owner, account.TypeUserWallet)
	cash := mustCreateAccount(t, accounts, owner, account.TypePlatformCash)
	original := mustPostReturning(t, transactions, cash.ID, wallet.ID, 300)

	errs := make(chan error, 2)

	for range 2 {
		go func() {
			_, err := transactions.Reverse(t.Context(), transaction.ReverseTransactionParams{
				ID: original.ID, Reference: "it-rev-" + uuid.NewString(), Reason: "race",
			})
			errs <- err
		}()
	}

	var won, lost int

	for range 2 {
		switch err := <-errs; {
		case err == nil:
			won++
		case errors.Is(err, transaction.ErrAlreadyReversed):
			lost++
		default:
			t.Fatalf("Reverse() unexpected error: %v", err)
		}
	}

	if won != 1 || lost != 1 {
		t.Errorf("won %d, lost %d; want exactly one of each", won, lost)
	}

	assertBalance(t, accounts, wallet.ID, 0)
}

func mustReverse(t *testing.T, repo transaction.Repository, id uuid.UUID) *transaction.Transaction {
	t.Helper()

	reversal, err := repo.Reverse(t.Context(), transaction.ReverseTransactionParams{
		ID: id, Reference: "it-rev-" + uuid.NewString(), Reason: itReason,
	})
	if err != nil {
		t.Fatalf("Reverse: %v", err)
	}

	return reversal
}

// checkMirror asserts reversal is a REVERSAL of original whose entries swap
// each original entry's side, account for account and amount for amount.
func checkMirror(t *testing.T, original, reversal *transaction.Transaction) {
	t.Helper()

	if reversal.Type != transaction.TypeReversal || reversal.ParentTransactionID.UUID != original.ID ||
		reversal.Currency != original.Currency || reversal.Description != itReason {
		t.Fatalf("reversal header = %+v, want REVERSAL of %s in %s", reversal, original.ID, original.Currency)
	}

	if len(reversal.Entries) != len(original.Entries) {
		t.Fatalf("reversal has %d entries, want %d", len(reversal.Entries), len(original.Entries))
	}

	for i, o := range original.Entries {
		r := reversal.Entries[i]
		if r.AccountID != o.AccountID || r.Amount != o.Amount || r.Type != o.Type.Opposite() {
			t.Errorf("entry %d = %+v, want the mirror of %+v", i, r, o)
		}
	}
}

// TestPostingRespectsAccountStatus posts into and out of a wallet in each
// status, and reverses into it, checking the gap-3 rules end to end.
func TestPostingRespectsAccountStatus(t *testing.T) {
	t.Parallel()

	accounts, transactions := newRepos(t)

	tests := []struct {
		status        account.Status
		creditOK      bool // cash -> wallet
		debitOK       bool // wallet -> cash
		reverseFundOK bool // reversing an earlier funding (debits the wallet)
	}{
		{status: account.StatusDebitRestricted, creditOK: true, debitOK: false, reverseFundOK: true},
		{status: account.StatusCreditRestricted, creditOK: false, debitOK: true, reverseFundOK: true},
		{status: account.StatusFullyFrozen, creditOK: false, debitOK: false, reverseFundOK: true},
		{status: account.StatusClosed, creditOK: false, debitOK: false, reverseFundOK: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()

			owner := "it-" + uuid.NewString()
			wallet := mustCreateAccount(t, accounts, owner, account.TypeUserWallet)
			cash := mustCreateAccount(t, accounts, owner, account.TypePlatformCash)
			funding := mustPostReturning(t, transactions, cash.ID, wallet.ID, 1_000)

			mustSetStatus(t, accounts, wallet.ID, tt.status)

			assertPostable(t, "credit", tt.creditOK, post(t, transactions, cash.ID, wallet.ID))
			assertPostable(t, "debit", tt.debitOK, post(t, transactions, wallet.ID, cash.ID))

			_, err := transactions.Reverse(t.Context(), transaction.ReverseTransactionParams{
				ID: funding.ID, Reference: "it-rev-" + uuid.NewString(), Reason: "status test",
			})
			assertPostable(t, "reversal", tt.reverseFundOK, err)
		})
	}

	t.Run("unknown account", func(t *testing.T) {
		t.Parallel()

		cash := mustCreateAccount(t, accounts, "it-"+uuid.NewString(), account.TypePlatformCash)

		if err := post(t, transactions, cash.ID, uuid.New()); !errors.Is(err, transaction.ErrEntryAccountNotFound) {
			t.Errorf("post to unknown account error = %v, want ErrEntryAccountNotFound", err)
		}
	})
}

// post is mustPost returning the error instead of failing.
func post(t *testing.T, repo transaction.Repository, debit, credit uuid.UUID) error {
	t.Helper()

	_, err := repo.Create(t.Context(), transaction.CreateTransactionParams{
		Reference:       "it-" + uuid.NewString(),
		Type:            itTxnType,
		Currency:        currency.NGN,
		Metadata:        []byte("{}"),
		TransactionDate: time.Now(),
		Entries: []transaction.EntryParams{
			{AccountID: debit, Type: domain.EntryTypeDebit, Amount: 10},
			{AccountID: credit, Type: domain.EntryTypeCredit, Amount: 10},
		},
	})

	return err
}

func assertPostable(t *testing.T, what string, wantOK bool, err error) {
	t.Helper()

	switch {
	case wantOK && err != nil:
		t.Errorf("%s: error = %v, want success", what, err)
	case !wantOK && !errors.Is(err, transaction.ErrAccountNotPostable):
		t.Errorf("%s: error = %v, want ErrAccountNotPostable", what, err)
	}
}

func mustSetStatus(t *testing.T, repo account.Repository, id uuid.UUID, status account.Status) {
	t.Helper()

	_, err := repo.ChangeAccountStatus(t.Context(), account.StatusChangeParams{
		AccountID: id,
		NewStatus: status,
		Reason:    itReason,
		ActorType: account.ActorTypeAdmin,
		ActorID:   "it",
	})
	if err != nil {
		t.Fatalf("set status %s: %v", status, err)
	}
}

// TestIntegrityTriggerErrors bypasses the service's checks to reach the
// database trigger, and checks its custom SQLSTATEs map to domain errors.
func TestIntegrityTriggerErrors(t *testing.T) {
	t.Parallel()

	accounts, transactions := newRepos(t)
	owner := "it-" + uuid.NewString()

	cash := mustCreateAccount(t, accounts, owner, account.TypePlatformCash) // NGN

	usdWallet, err := accounts.Create(t.Context(), account.CreateAccountParams{
		OwnerID: owner, Type: string(account.TypeUserWallet), Currency: currency.USD.String(),
	})
	if err != nil {
		t.Fatalf("create USD wallet: %v", err)
	}

	ngnWallet := mustCreateAccount(t, accounts, owner, account.TypeUserWallet)

	tests := []struct {
		name    string
		entries []transaction.EntryParams
		want    error
	}{
		{
			name: "currency mismatch",
			entries: []transaction.EntryParams{
				{AccountID: cash.ID, Type: domain.EntryTypeDebit, Amount: 10},
				{AccountID: usdWallet.ID, Type: domain.EntryTypeCredit, Amount: 10},
			},
			want: transaction.ErrCurrencyMismatch,
		},
		{
			name: "unbalanced",
			entries: []transaction.EntryParams{
				{AccountID: cash.ID, Type: domain.EntryTypeDebit, Amount: 10},
				{AccountID: ngnWallet.ID, Type: domain.EntryTypeCredit, Amount: 9},
			},
			want: transaction.ErrUnbalancedTransaction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := transactions.Create(t.Context(), transaction.CreateTransactionParams{
				Reference:       "it-" + uuid.NewString(),
				Type:            itTxnType,
				Currency:        currency.NGN,
				Metadata:        []byte("{}"),
				TransactionDate: time.Now(),
				Entries:         tt.entries,
			})
			if !errors.Is(err, tt.want) {
				t.Errorf("Create() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestStatementSummary_EndIsExclusive checks the summary's window is half-open:
// an entry at exactly end_date is outside it, as for statement entries.
func TestStatementSummary_EndIsExclusive(t *testing.T) {
	t.Parallel()

	accounts, transactions := newRepos(t)
	owner := "it-" + uuid.NewString()
	wallet := mustCreateAccount(t, accounts, owner, account.TypeUserWallet)
	cash := mustCreateAccount(t, accounts, owner, account.TypePlatformCash)

	funding := mustPostReturning(t, transactions, cash.ID, wallet.ID, 500)
	at := funding.Entries[0].CreatedAt

	for _, tc := range []struct {
		name    string
		end     time.Time
		credits int64
	}{
		{name: "end at the entry excludes it", end: at, credits: 0},
		{name: "end just after includes it", end: at.Add(time.Microsecond), credits: 500},
	} {
		summary, err := accounts.GetStatementSummary(t.Context(), account.GetStatementSummaryParams{
			AccountID: wallet.ID, StartDate: at.Add(-time.Hour), EndDate: tc.end,
		})
		if err != nil {
			t.Fatalf("%s: GetStatementSummary: %v", tc.name, err)
		}

		if summary.TotalCredits != tc.credits {
			t.Errorf("%s: total credits = %d, want %d", tc.name, summary.TotalCredits, tc.credits)
		}
	}
}
