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
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
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

func (f *fakeAccounts) ListAccounts(_ context.Context, p account.ListAccountsParams) ([]account.Account, error) {
	list := []account.Account{{ID: uuid.MustParse(testID), CreatedAt: fixedTime}}
	return list, f.record(p, list)
}

func (f *fakeAccounts) GetAccountBalance(_ context.Context, id uuid.UUID) (*account.GetAccountBalanceResult, error) {
	b := &account.GetAccountBalanceResult{Currency: ngn, Exponent: 2, BalanceMinor: 1050, BalanceMajor: "10.50"}
	return b, f.record(id, b)
}

func (f *fakeAccounts) GetAccountStatementSummary(_ context.Context, p account.GetStatementSummaryParams) (*account.StatementSummary, error) {
	s := &account.StatementSummary{AccountID: p.AccountID, ClosingBalance: 7}
	return s, f.record(p, s)
}

func (f *fakeAccounts) GetAccountStatementEntries(_ context.Context, p account.GetStatementEntriesParams) ([]account.StatementEntry, error) {
	e := []account.StatementEntry{{ID: uuid.MustParse(testID), Amount: 3, CreatedAt: fixedTime}}
	return e, f.record(p, e)
}

func (f *fakeAccounts) ReconcileAccount(_ context.Context, id uuid.UUID) (*account.ReconciliationResult, error) {
	r := &account.ReconciliationResult{AccountID: id, Status: account.ReconciliationStatusReconciled, ReconciledAt: fixedTime}
	return r, f.record(id, r)
}

func (f *fakeAccounts) CreateAccountSnapshot(_ context.Context, p account.CreateAccountSnapshotParams) (*account.BalanceSnapshot, error) {
	s := &account.BalanceSnapshot{AccountID: p.ID, Balance: 9, CreatedAt: fixedTime}
	return s, f.record(p, s)
}

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

func (f *fakeTransactions) CreateTransaction(_ context.Context, p transaction.CreateTransactionParams) (*transaction.Transaction, error) {
	t := &transaction.Transaction{ID: uuid.MustParse(testID), Reference: p.Reference, CreatedAt: fixedTime}
	return t, f.record(p, t)
}

func (f *fakeTransactions) GetTransactionDetails(_ context.Context, id string) (*transaction.Transaction, error) {
	t := &transaction.Transaction{ID: uuid.MustParse(id), CreatedAt: fixedTime}
	return t, f.record(id, t)
}

func (f *fakeTransactions) ListTransactions(_ context.Context, p transaction.ListTransactionsParams) ([]transaction.Transaction, error) {
	list := []transaction.Transaction{{ID: uuid.MustParse(testID), CreatedAt: fixedTime}}
	return list, f.record(p, list)
}

func (f *fakeTransactions) ReverseTransaction(_ context.Context, id string) error {
	return f.record(id, nil)
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
	return handler.NewHandler(accounts, txns, slog.New(slog.DiscardHandler))
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
		{name: "unexpected error", serviceErr: errors.New("db down"), wantStatus: http.StatusInternalServerError, wantCalled: true},
	}
}
