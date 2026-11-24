package handler_test

import (
	"cmp"
	"encoding/json"
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

const (
	otherID     = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5c"
	reverseBody = `{"reference":"rev-1","reason":"duplicate"}`
)

// TestTransactionHandlers pins each transaction endpoint's observable contract,
// as TestAccountHandlers does for accounts.
func TestTransactionHandlers(t *testing.T) {
	t.Parallel()

	accountID := uuid.MustParse(testID)
	createBody := `{"reference":"r1","type":"FUND","currency":"NGN","description":"d","metadata":{"order":7},` +
		`"transaction_date":"2026-01-02T03:04:05Z","entries":[` +
		`{"account_id":"` + testID + `","type":"DEBIT","amount":5},` +
		`{"account_id":"` + testID + `","type":"CREDIT","amount":5}]}`
	createParams := transaction.CreateTransactionParams{
		Reference:       "r1",
		Type:            "FUND",
		Currency:        currency.NGN,
		Description:     "d",
		Metadata:        json.RawMessage(`{"order":7}`),
		TransactionDate: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Entries: []transaction.EntryParams{
			{AccountID: accountID, Type: domain.EntryTypeDebit, Amount: 5},
			{AccountID: accountID, Type: domain.EntryTypeCredit, Amount: 5},
		},
	}

	tests := []txnCase{
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
			wantParams: accountID,
		},
		{
			name:       "details rejects malformed ID",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.GetTransactionDetails },
			req:        request{method: http.MethodGet, target: txnPath + "/x", pathID: "x"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "list",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.ListTransactions },
			req: request{
				method: http.MethodGet,
				target: txnPath + "?account_id=" + testID + "&reference=r1&type=FUND&currency=NGN" +
					"&start_date=2026-01-01T00:00:00Z&end_date=2026-02-01T00:00:00Z&cursor=" + testID + "&limit=10",
			},
			wantStatus: http.StatusOK,
			wantParams: transaction.ListTransactionsParams{
				AccountID: uuid.NullUUID{UUID: accountID, Valid: true},
				Reference: "r1",
				Type:      "FUND",
				Currency:  currency.NGN,
				StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				Cursor:    uuid.NullUUID{UUID: accountID, Valid: true},
				Limit:     10,
			},
		},
		{
			name:       "list rejects lowercase currency",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.ListTransactions },
			req:        request{method: http.MethodGet, target: txnPath + "?currency=ngn"},
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
	}

	runTxnCases(t, tests)
}

// txnCase pins one transaction endpoint call, as accountCase does for accounts.
type txnCase struct {
	name        string
	endpoint    func(handler.Handler) http.HandlerFunc
	req         request
	serviceErr  error
	wantStatus  int
	wantMessage string
	wantParams  any
	wantNoBody  bool
}

func runTxnCases(t *testing.T, tests []txnCase) {
	t.Helper()

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

func TestTransactionHandlers_ReverseAndConvert(t *testing.T) {
	t.Parallel()

	accountID := uuid.MustParse(testID)

	runTxnCases(t, []txnCase{
		{
			name:     "reverse",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.ReverseTransaction },
			req: request{
				method: http.MethodPost,
				target: txnPath + "/" + testID + "/reverse",
				pathID: testID,
				body:   reverseBody,
			},
			wantStatus:  http.StatusCreated,
			wantMessage: "transaction reversed",
			wantParams:  transaction.ReverseTransactionParams{ID: accountID, Reference: "rev-1", Reason: "duplicate"},
		},
		{
			name:       "reverse requires reference and reason",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.ReverseTransaction },
			req:        request{method: http.MethodPost, target: "/x", pathID: testID, body: `{}`},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "reverse maps already reversed",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.ReverseTransaction },
			req:        request{method: http.MethodPost, target: "/x", pathID: testID, body: reverseBody},
			serviceErr: transaction.ErrAlreadyReversed,
			wantStatus: http.StatusConflict,
			wantParams: transaction.ReverseTransactionParams{ID: accountID, Reference: "rev-1", Reason: "duplicate"},
		},
		{
			name:     "conversion",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.CreateConversion },
			req: request{method: http.MethodPost, target: "/conversions", body: `{"reference":"fx-1",` +
				`"source_account_id":"` + testID + `","target_account_id":"` + otherID + `",` +
				`"source_amount":100,"rate":"0.00065"}`},
			wantStatus:  http.StatusCreated,
			wantMessage: "conversion created",
			wantParams: transaction.CreateConversionParams{
				Reference:       "fx-1",
				SourceAccountID: accountID,
				TargetAccountID: uuid.MustParse(otherID),
				SourceAmount:    100,
				Rate:            "0.00065",
			},
		},
	})
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
					request{method: http.MethodPost, target: "/x", pathID: cmp.Or(ec.pathID, testID), body: reverseBody},
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
