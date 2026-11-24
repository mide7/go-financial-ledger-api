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
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
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

type fakeTransactions struct{ transaction.Service }

func (fakeTransactions) GetTransactionDetails(context.Context, string) (*transaction.Transaction, error) {
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
		handler.NewHandler(accounts, fakeTransactions{}, slog.New(slog.NewJSONHandler(&logs, nil))),
		handler.NewHealthCheckHandler(nil), // health routes are not exercised here,
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
		{
			name:   "unimplemented transaction lookup",
			method: http.MethodGet,
			target: "/api/v1/transactions/" + accountID,
			want:   http.StatusNotImplemented,
		},
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
