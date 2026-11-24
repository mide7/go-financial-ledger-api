package http_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/domain/withdrawal"
	httptransport "github.com/mide7/go-financial-ledger-api/internal/platform/transport/http"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
)

const accountID = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"

// fakeAccounts embeds the interface so unexercised methods panic if called.
type fakeAccounts struct {
	account.Service

	summary  account.GetStatementSummaryParams
	snapshot account.CreateAccountSnapshotParams
}

func (f *fakeAccounts) GetAccountDetails(_ context.Context, _ uuid.UUID) (*account.Account, error) {
	return nil, account.ErrAccountNotFound
}

// GetAccountBalance fails with an error the HTTP layer does not recognize,
// standing in for a database outage.
func (f *fakeAccounts) GetAccountBalance(context.Context, uuid.UUID) (*account.GetAccountBalanceResult, error) {
	return nil, errors.New("pq: connection refused")
}

func (f *fakeAccounts) GetAccountStatementSummary(
	_ context.Context,
	p account.GetStatementSummaryParams,
) (*account.StatementSummary, error) {
	f.summary = p
	return &account.StatementSummary{AccountID: p.AccountID}, nil
}

func (f *fakeAccounts) CreateAccountSnapshot(
	_ context.Context,
	p account.CreateAccountSnapshotParams,
) (*account.BalanceSnapshot, error) {
	f.snapshot = p
	return &account.BalanceSnapshot{AccountID: p.ID}, nil
}

func (f *fakeAccounts) ListUnresolvedDiscrepancies(context.Context) ([]account.DiscrepancyRecord, error) {
	return []account.DiscrepancyRecord{}, nil
}

// fakeCurrencies embeds the interface so unexercised methods panic if called.
type fakeCurrencies struct{ currency.Registry }

func (fakeCurrencies) List() []currency.Currency {
	return []currency.Currency{{Code: currency.NGN, Name: "Naira", Exponent: 2, IsActive: true}}
}

type fakeTransactions struct{ transaction.Service }

func (fakeTransactions) GetTransactionDetails(context.Context, uuid.UUID) (*transaction.Transaction, error) {
	return nil, domain.ErrNotImplemented
}

func serve(t *testing.T, accounts *fakeAccounts, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	rec, _ := serveWithLogs(t, accounts, method, target, body)

	return rec
}

// serveWithLogs is serve plus everything the handler logged, as JSON lines.
func serveWithLogs(
	t *testing.T,
	accounts *fakeAccounts,
	method, target, body string,
) (*httptest.ResponseRecorder, string) {
	t.Helper()

	var logs bytes.Buffer

	router := httptransport.NewRouter(
		handler.NewHandler(handler.Services{
			Accounts:     accounts,
			Transactions: fakeTransactions{},
			Withdrawals:  withdrawal.NewWithdrawalService(),
			Currencies:   fakeCurrencies{},
		}, slog.New(slog.NewJSONHandler(&logs, nil))),
		handler.NewHealthCheckHandler(nil), // health routes are not exercised here,
		nil,                                // nor idempotency (see middleware tests)
	)

	var req *http.Request
	if body == "" {
		req = httptest.NewRequestWithContext(t.Context(), method, target, nil)
	} else {
		req = httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec, logs.String()
}

func TestStatementSummary_ReadsPathAndQuery(t *testing.T) {
	t.Parallel()

	accounts := &fakeAccounts{}
	rec := serve(t, accounts, http.MethodGet,
		"/api/v1/accounts/"+
			accountID+
			"/statement/summary?start_date=2026-01-01T00:00:00Z&end_date=2026-02-01T00:00:00Z", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}

	if accounts.summary.AccountID.String() != accountID {
		t.Errorf("account ID = %s, want %s", accounts.summary.AccountID, accountID)
	}

	if want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC); !accounts.summary.StartDate.Equal(want) {
		t.Errorf("start date = %v, want %v", accounts.summary.StartDate, want)
	}
}

func TestSnapshot_ReadsPathAndBody(t *testing.T) {
	t.Parallel()

	accounts := &fakeAccounts{}
	rec := serve(t, accounts, http.MethodPost, "/api/v1/accounts/"+accountID+"/snapshot", `{"reason":"end of day"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
	}

	if accounts.snapshot.Reason != "end of day" || accounts.snapshot.ID.String() != accountID {
		t.Errorf("params = %+v", accounts.snapshot)
	}
}

func TestStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, method, target, body string
		want                       int
	}{
		{
			name:   "malformed account ID",
			method: http.MethodGet,
			target: "/api/v1/accounts/not-a-uuid",
			want:   http.StatusUnprocessableEntity,
		},
		{
			name:   "unknown account",
			method: http.MethodGet,
			target: "/api/v1/accounts/" + accountID,
			want:   http.StatusNotFound,
		},
		{
			name: "unsupported snapshot field", method: http.MethodPost, target: "/api/v1/accounts/" +
				accountID +
				"/snapshot",
			body: `{"reason":"x","target_entry_sequence":5}`, want: http.StatusUnprocessableEntity,
		},
		{
			name: "unsupported reconcile field", method: http.MethodPost, target: "/api/v1/accounts/" +
				accountID +
				"/reconcile",
			body: `{"as_of_timestamp":"2026-01-01T00:00:00Z"}`, want: http.StatusUnprocessableEntity,
		},
		// The new routes resolve: a malformed ID reaches the handler's
		// decoder (422) instead of the mux (404/405).
		{
			name:   "latest snapshot route",
			method: http.MethodGet,
			target: "/api/v1/accounts/not-a-uuid/snapshot",
			want:   http.StatusUnprocessableEntity,
		},
		{
			name:   "status change route",
			method: http.MethodPatch,
			target: "/api/v1/accounts/not-a-uuid/status",
			want:   http.StatusUnprocessableEntity,
		},
		{
			name:   "status history route",
			method: http.MethodGet,
			target: "/api/v1/accounts/not-a-uuid/status/history",
			want:   http.StatusUnprocessableEntity,
		},
		{
			name:   "resolve discrepancy route",
			method: http.MethodPost,
			target: "/api/v1/discrepancies/not-a-uuid/resolve",
			want:   http.StatusUnprocessableEntity,
		},
		{name: "discrepancies route", method: http.MethodGet, target: "/api/v1/discrepancies", want: http.StatusOK},
		{name: "currencies route", method: http.MethodGet, target: "/api/v1/currencies", want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if rec := serve(t, &fakeAccounts{}, tt.method, tt.target, tt.body); rec.Code != tt.want {
				t.Errorf("status = %d, want %d; body: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

// TestUnexpectedErrors_AreLogged checks the split between client and log: an
// unrecognized error reaches the client only as a generic 500, so its cause
// must be logged; a known error (404) must not be logged as a failure.
func TestUnexpectedErrors_AreLogged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantLogged bool
	}{
		{
			name:       "unexpected error",
			target:     "/api/v1/accounts/" + accountID + "/balance",
			wantStatus: http.StatusInternalServerError,
			wantLogged: true,
		},
		{
			name:       "known error",
			target:     "/api/v1/accounts/" + accountID,
			wantStatus: http.StatusNotFound,
			wantLogged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec, logs := serveWithLogs(t, &fakeAccounts{}, http.MethodGet, tt.target, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body)
			}

			if strings.Contains(rec.Body.String(), "connection refused") {
				t.Error("internal error detail leaked to the client")
			}

			logged := strings.Contains(logs, `"msg":"request failed"`) && strings.Contains(logs, "connection refused")
			if logged != tt.wantLogged {
				t.Errorf("failure logged = %v, want %v; logs: %s", logged, tt.wantLogged, logs)
			}
		})
	}
}

// TestScaffoldedRoutes_NotImplemented sends a valid request to every route
// whose service is still a stub, through the real stub services, and expects
// 501. Remove a case when its route is implemented (see docs/todo.md).
func TestScaffoldedRoutes_NotImplemented(t *testing.T) {
	t.Parallel()

	discard := slog.New(slog.DiscardHandler)
	router := httptransport.NewRouter(
		handler.NewHandler(handler.Services{
			Accounts:     account.NewAccountService(nil, nil, discard),
			Transactions: transaction.NewTransactionService(nil, nil),
			Withdrawals:  withdrawal.NewWithdrawalService(),
			Currencies:   fakeCurrencies{},
		}, discard),
		handler.NewHealthCheckHandler(nil),
		nil,
	)

	tests := []struct{ method, target, body string }{
		{
			method: http.MethodPost,
			target: "/api/v1/conversions",
			body: `{"reference":"fx-1","source_account_id":"` + accountID +
				`","target_account_id":"` + uuid.NewString() + `","source_amount":100,"rate":"1.5"}`,
		},
		{method: http.MethodPost, target: "/api/v1/reconciliations"},
		{
			method: http.MethodPost,
			target: "/api/v1/withdrawals",
			body:   `{"account_id":"` + accountID + `","amount":100,"processor":"PAYSTACK","reference":"wd-1"}`,
		},
		{method: http.MethodGet, target: "/api/v1/withdrawals"},
		{method: http.MethodGet, target: "/api/v1/withdrawals/" + accountID},
		{
			method: http.MethodPost,
			target: "/api/v1/withdrawals/" + accountID + "/settle",
			body:   `{"processor_reference":"psk-1"}`,
		},
		{method: http.MethodPost, target: "/api/v1/withdrawals/" + accountID + "/fail", body: `{"reason":"bank down"}`},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			t.Parallel()

			var body *strings.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			var req *http.Request
			if body == nil {
				req = httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, nil)
			} else {
				req = httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, body)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotImplemented {
				t.Errorf("status = %d, want 501; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

// TestRegisterV1Routes_IdempotentRoutes checks that exactly the POST routes
// that move money pass through the idempotent middleware.
func TestRegisterV1Routes_IdempotentRoutes(t *testing.T) {
	t.Parallel()

	const marker = "X-Test-Idempotent"

	// mark answers by itself, so wrapped routes never reach the fake services.
	mark := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(marker, "yes")
			w.WriteHeader(http.StatusNoContent)
		})
	}

	discard := slog.New(slog.DiscardHandler)
	router := httptransport.RegisterV1Routes(handler.NewHandler(handler.Services{
		Accounts:     &fakeAccounts{},
		Transactions: fakeTransactions{},
		Withdrawals:  withdrawal.NewWithdrawalService(),
		Currencies:   fakeCurrencies{},
	}, discard), mark)

	tests := []struct {
		method, target string
		want           bool
	}{
		{method: http.MethodPost, target: "/transactions", want: true},
		{method: http.MethodPost, target: "/transactions/" + accountID + "/reverse", want: true},
		{method: http.MethodPost, target: "/conversions", want: true},
		{method: http.MethodPost, target: "/withdrawals", want: true},
		{method: http.MethodGet, target: "/transactions"},
		{method: http.MethodPost, target: "/accounts"},
		{method: http.MethodPost, target: "/withdrawals/" + accountID + "/settle"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()

			func() {
				// Unwrapped routes reach the fakes, whose unstubbed methods panic.
				defer func() { _ = recover() }()

				router.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, strings.NewReader("{}")))
			}()

			if got := rec.Header().Get(marker) == "yes"; got != tt.want {
				t.Errorf("wrapped = %v, want %v", got, tt.want)
			}
		})
	}
}
