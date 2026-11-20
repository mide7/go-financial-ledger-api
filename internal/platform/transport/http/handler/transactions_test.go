package handler_test

import (
	"cmp"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
)

// TestTransactionHandlers pins each transaction endpoint's observable contract,
// as TestAccountHandlers does for accounts.
func TestTransactionHandlers(t *testing.T) {
	t.Parallel()

	accountID := uuid.MustParse(testID)
	createBody := `{"reference":"r1","type":"FUND","currency":"NGN","description":"d","metadata":"e30=",` +
		`"transaction_date":"2026-01-02T03:04:05Z","entries":[` +
		`{"account_id":"` + testID + `","type":"DEBIT","amount":5},` +
		`{"account_id":"` + testID + `","type":"CREDIT","amount":5}]}`
	createParams := transaction.CreateTransactionParams{
		Reference:       "r1",
		Type:            "FUND",
		Currency:        currency.NGN,
		Description:     "d",
		Metadata:        []byte("{}"),
		TransactionDate: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Entries: []transaction.EntryParams{
			{AccountID: accountID, Type: domain.EntryTypeDebit, Amount: 5},
			{AccountID: accountID, Type: domain.EntryTypeCredit, Amount: 5},
		},
	}

	tests := []struct {
		name        string
		endpoint    func(handler.Handler) http.HandlerFunc
		req         request
		serviceErr  error
		wantStatus  int
		wantMessage string
		wantParams  any
		wantNoBody  bool
	}{
		{
			name:        "create",
			endpoint:    func(h handler.Handler) http.HandlerFunc { return h.CreateTransaction },
			req:         request{method: http.MethodPost, target: txnPath, body: createBody},
			wantStatus:  http.StatusCreated,
			wantMessage: "transaction created",
			wantParams:  createParams,
		},
		{
			name:       "create rejects malformed JSON",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.CreateTransaction },
			req:        request{method: http.MethodPost, target: txnPath, body: `{`},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "create maps domain error",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.CreateTransaction },
			req:        request{method: http.MethodPost, target: txnPath, body: createBody},
			serviceErr: transaction.ErrDuplicateReference,
			wantStatus: http.StatusUnprocessableEntity,
			wantParams: createParams,
		},
		{
			name:       "transaction details",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.GetTransactionDetails },
			req:        request{method: http.MethodGet, target: txnPath + "/" + testID, pathID: testID},
			wantStatus: http.StatusOK,
			wantParams: testID,
		},
		{
			name:       "details rejects malformed ID",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.GetTransactionDetails },
			req:        request{method: http.MethodGet, target: txnPath + "/x", pathID: "x"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "list",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.ListTransactions },
			req:        request{method: http.MethodGet, target: txnPath + "?page=3&limit=10&type=FUND&status=POSTED"},
			wantStatus: http.StatusOK,
			wantParams: transaction.ListTransactionsParams{Page: 3, Limit: 10, Type: "FUND", Status: "POSTED"},
		},
		{
			name:       "list rejects unknown status",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.ListTransactions },
			req:        request{method: http.MethodGet, target: txnPath + "?status=LOST"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "list maps not implemented",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.ListTransactions },
			req:        request{method: http.MethodGet, target: txnPath},
			serviceErr: domain.ErrNotImplemented,
			wantStatus: http.StatusNotImplemented,
			wantParams: transaction.ListTransactionsParams{},
		},
		{
			name:        "reverse",
			endpoint:    func(h handler.Handler) http.HandlerFunc { return h.ReverseTransaction },
			req:         request{method: http.MethodPost, target: txnPath + "/" + testID + "/reverse", pathID: testID},
			wantStatus:  http.StatusOK,
			wantMessage: "transaction reversed",
			wantParams:  testID,
			wantNoBody:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txns := &fakeTransactions{err: tt.serviceErr}
			got := do(t, tt.endpoint(newHandler(&fakeAccounts{}, txns)), tt.req)

			assertResponse(t, got, tt.wantStatus, tt.wantMessage)
			assertParams(t, txns.got, tt.wantParams)

			switch {
			case tt.wantNoBody:
				if got.body != "" {
					t.Errorf("body = %q, want none", got.body)
				}
			case tt.serviceErr == nil && tt.wantParams != nil:
				if want := encoded(t, txns.resp); got.body != want {
					t.Errorf("body = %s, want %s", got.body, want)
				}
			}
		})
	}
}

// TestTransactionHandlers_Errors runs the path-ID transaction endpoints
// through the error matrix.
func TestTransactionHandlers_Errors(t *testing.T) {
	t.Parallel()

	endpoints := []struct {
		name     string
		endpoint func(handler.Handler) http.HandlerFunc
	}{
		{name: "details", endpoint: func(h handler.Handler) http.HandlerFunc { return h.GetTransactionDetails }},
		{name: "reverse", endpoint: func(h handler.Handler) http.HandlerFunc { return h.ReverseTransaction }},
	}

	for _, ep := range endpoints {
		for _, ec := range errorMatrix(transaction.ErrTransactionNotFound) {
			t.Run(ep.name+"/"+ec.name, func(t *testing.T) {
				t.Parallel()

				txns := &fakeTransactions{err: ec.serviceErr}
				got := do(
					t,
					ep.endpoint(newHandler(&fakeAccounts{}, txns)),
					request{method: http.MethodPost, target: "/x", pathID: cmp.Or(ec.pathID, testID)},
				)

				assertResponse(t, got, ec.wantStatus, "")

				if called := txns.got != nil; called != ec.wantCalled {
					t.Errorf("service called = %v, want %v", called, ec.wantCalled)
				}
			})
		}
	}
}

func TestListTransactions_ServiceError(t *testing.T) {
	t.Parallel()

	txns := &fakeTransactions{err: errors.New("db down")}
	got := do(t, newHandler(&fakeAccounts{}, txns).ListTransactions, request{method: http.MethodGet, target: txnPath})

	assertResponse(t, got, http.StatusInternalServerError, "")
}
