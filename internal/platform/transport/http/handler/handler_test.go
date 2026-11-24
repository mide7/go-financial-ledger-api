package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/domain/withdrawal"
	"github.com/mide7/go-financial-ledger-api/internal/pagination"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

const (
	testID  = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	ngn     = "NGN"
	txnPath = "/transactions"
)

var fixedTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// fakeAccounts records the params each method receives and returns err when set.
type fakeAccounts struct {
	err  error
	got  any
	resp any
}

func (f *fakeAccounts) record(p, resp any) error {
	f.got, f.resp = p, resp
	return f.err
}

func (f *fakeAccounts) CreateAccount(_ context.Context, p account.CreateAccountParams) (*account.Account, error) {
	a := &account.Account{ID: uuid.MustParse(testID), OwnerID: p.OwnerID, CreatedAt: fixedTime}
	return a, f.record(p, a)
}

func (f *fakeAccounts) GetAccountDetails(_ context.Context, id uuid.UUID) (*account.Account, error) {
	a := &account.Account{ID: id, CreatedAt: fixedTime}
	return a, f.record(id, a)
}

func (f *fakeAccounts) ListAccounts(
	_ context.Context,
	p account.ListAccountsParams,
) (pagination.OffsetPaginatedResult[account.Account], error) {
	list := []account.Account{{ID: uuid.MustParse(testID), CreatedAt: fixedTime}}
	page := pagination.NewOffsetPaginatedResult(list, 1, 20, 1)

	return page, f.record(p, page)
}

func (f *fakeAccounts) GetAccountBalance(_ context.Context, id uuid.UUID) (*account.GetAccountBalanceResult, error) {
	b := &account.GetAccountBalanceResult{Currency: ngn, Exponent: 2, BalanceMinor: 1050, BalanceMajor: "10.50"}
	return b, f.record(id, b)
}

func (f *fakeAccounts) GetAccountStatementSummary(
	_ context.Context,
	p account.GetStatementSummaryParams,
) (*account.StatementSummary, error) {
	s := &account.StatementSummary{AccountID: p.AccountID, ClosingBalance: 7}
	return s, f.record(p, s)
}

func (f *fakeAccounts) GetAccountStatementEntries(
	_ context.Context,
	p account.GetStatementEntriesParams,
) (pagination.CursorPaginatedResult[account.StatementEntry], error) {
	e := account.StatementEntry{ID: uuid.MustParse(testID), Amount: 3, CreatedAt: fixedTime}
	page := cursorPage(e, e.ID)

	return page, f.record(p, page)
}

func (f *fakeAccounts) ReconcileAccount(_ context.Context, id uuid.UUID) (*account.ReconciliationResult, error) {
	r := &account.ReconciliationResult{
		AccountID:    id,
		Status:       account.ReconciliationStatusReconciled,
		ReconciledAt: fixedTime,
	}

	return r, f.record(id, r)
}

func (f *fakeAccounts) CreateAccountSnapshot(
	_ context.Context,
	p account.CreateAccountSnapshotParams,
) (*account.BalanceSnapshot, error) {
	s := &account.BalanceSnapshot{AccountID: p.ID, Balance: 9, CreatedAt: fixedTime}
	return s, f.record(p, s)
}

func (f *fakeAccounts) GetLatestAccountSnapshot(_ context.Context, id uuid.UUID) (*account.BalanceSnapshot, error) {
	s := &account.BalanceSnapshot{AccountID: id, Balance: 4, CreatedAt: fixedTime}
	return s, f.record(id, s)
}

func (f *fakeAccounts) ChangeAccountStatus(
	_ context.Context,
	p account.StatusChangeParams,
) (*account.StatusHistory, error) {
	h := &account.StatusHistory{AccountID: p.AccountID, NewStatus: p.NewStatus, CreatedAt: fixedTime}
	return h, f.record(p, h)
}

func (f *fakeAccounts) ListAccountStatusHistory(
	_ context.Context,
	p account.ListAccountStatusHistoryParams,
) (pagination.CursorPaginatedResult[account.StatusHistory], error) {
	h := account.StatusHistory{AccountID: p.AccountID, NewStatus: account.StatusActive, CreatedAt: fixedTime}
	page := cursorPage(h, uuid.MustParse(testID))

	return page, f.record(p, page)
}

func (f *fakeAccounts) ReconcileAllAccounts(context.Context) (*account.ReconciliationRun, error) {
	r := &account.ReconciliationRun{AccountsChecked: 3, StartedAt: fixedTime, FinishedAt: fixedTime}
	return r, f.record(struct{}{}, r)
}

func (f *fakeAccounts) ListUnresolvedDiscrepancies(
	_ context.Context,
	p account.ListDiscrepanciesParams,
) (pagination.CursorPaginatedResult[account.DiscrepancyRecord], error) {
	d := account.DiscrepancyRecord{ID: uuid.MustParse(testID), DiscrepancyAmount: 2, DetectedAt: fixedTime}
	page := cursorPage(d, d.ID)

	return page, f.record(p, page)
}

// cursorPage wraps item as a one-item page with more to come, so handler
// tests see every page field, next_cursor included, round-trip through JSON.
func cursorPage[T any](item T, id uuid.UUID) pagination.CursorPaginatedResult[T] {
	return pagination.NewCursorPaginatedResult([]T{item, item}, 1, func(T) pagination.Cursor {
		return pagination.Cursor{ID: id}
	})
}

// resolveCall is what fakeAccounts.ResolveDiscrepancy records.
type resolveCall struct {
	ID    uuid.UUID
	Notes string
}

func (f *fakeAccounts) ResolveDiscrepancy(_ context.Context, id uuid.UUID, notes string) error {
	return f.record(resolveCall{ID: id, Notes: notes}, nil)
}

// fakeCurrencies is a currency.Registry holding a fixed list.
type fakeCurrencies struct{ list []currency.Currency }

func (f fakeCurrencies) Get(code currency.Code) (currency.Currency, bool) {
	for _, c := range f.list {
		if c.Code == code {
			return c, true
		}
	}

	return currency.Currency{}, false
}

func (f fakeCurrencies) List() []currency.Currency { return f.list }

func (fakeCurrencies) Reload(context.Context) error { return nil }

// fakeTransactions mirrors fakeAccounts for the transaction service.
type fakeTransactions struct {
	err  error
	got  any
	resp any
}

func (f *fakeTransactions) record(p, resp any) error {
	f.got, f.resp = p, resp
	return f.err
}

func (f *fakeTransactions) CreateTransaction(
	_ context.Context,
	p transaction.CreateTransactionParams,
) (*transaction.Transaction, error) {
	t := &transaction.Transaction{ID: uuid.MustParse(testID), Reference: p.Reference, CreatedAt: fixedTime}
	return t, f.record(p, t)
}

func (f *fakeTransactions) GetTransactionDetails(_ context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	t := &transaction.Transaction{ID: id, CreatedAt: fixedTime}
	return t, f.record(id, t)
}

func (f *fakeTransactions) ListTransactions(
	_ context.Context,
	p transaction.ListTransactionsParams,
) (pagination.CursorPaginatedResult[transaction.Transaction], error) {
	txn := transaction.Transaction{ID: uuid.MustParse(testID), CreatedAt: fixedTime}
	page := cursorPage(txn, txn.ID)

	return page, f.record(p, page)
}

func (f *fakeTransactions) ReverseTransaction(
	_ context.Context,
	p transaction.ReverseTransactionParams,
) (*transaction.Transaction, error) {
	t := &transaction.Transaction{ID: uuid.MustParse(testID), Reference: p.Reference, CreatedAt: fixedTime}
	return t, f.record(p, t)
}

func (f *fakeTransactions) CreateConversion(
	_ context.Context,
	p transaction.CreateConversionParams,
) (*transaction.Conversion, error) {
	c := &transaction.Conversion{Reference: p.Reference, Rate: p.Rate}
	return c, f.record(p, c)
}

// request describes one call to a handler method.
type request struct {
	method string
	target string
	pathID string
	body   string
}

// response is everything a handler observably produced.
type response struct {
	status     int
	hasMessage bool
	message    string
	body       string
}

func do(t *testing.T, h http.HandlerFunc, req request) response {
	t.Helper()

	var r *http.Request
	if req.body == "" {
		r = httptest.NewRequestWithContext(t.Context(), req.method, req.target, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), req.method, req.target, strings.NewReader(req.body))
	}

	if req.pathID != "" {
		r.SetPathValue("id", req.pathID)
	}

	rec := httptest.NewRecorder()
	h(rec, r)

	msgs, hasMessage := rec.Header()[http.CanonicalHeaderKey(middleware.HeaderResponseMessage)]

	resp := response{status: rec.Code, hasMessage: hasMessage, body: rec.Body.String()}
	if hasMessage {
		resp.message = msgs[0]
	}

	return resp
}

// encoded is v exactly as json.Encoder writes it, trailing newline included.
func encoded(t *testing.T, v any) string {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return string(b) + "\n"
}

func newHandler(accounts *fakeAccounts, txns *fakeTransactions) handler.Handler {
	return handler.NewHandler(handler.Services{
		Accounts:     accounts,
		Transactions: txns,
		Withdrawals:  withdrawal.NewWithdrawalService(),
		Currencies:   fakeCurrencies{},
	}, slog.New(slog.DiscardHandler))
}

// errorCase is one way an endpoint can fail and the status it must produce.
type errorCase struct {
	name       string
	pathID     string // overrides the endpoint's path ID when set
	serviceErr error
	wantStatus int
	wantCalled bool
}

// errorMatrix applies to every endpoint that reads an ID from the path.
func errorMatrix(notFound error) []errorCase {
	return []errorCase{
		{name: "malformed path ID", pathID: "not-a-uuid", wantStatus: http.StatusUnprocessableEntity},
		{name: "known domain error", serviceErr: notFound, wantStatus: http.StatusNotFound, wantCalled: true},
		{
			name:       "unexpected error",
			serviceErr: errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantCalled: true,
		},
	}
}
