package handler_test

import (
	"cmp"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
)

const (
	accountsPath   = "/accounts"
	statementRange = "start_date=2026-01-01T00:00:00Z&end_date=2026-02-01T00:00:00Z"
	validAccount   = `{"owner_id":"u1","type":"USER_WALLET","currency":"NGN"}`
)

// accountCase pins one account endpoint call: which request parts it reads,
// the params it passes on, and the exact status, message header and body.
type accountCase struct {
	name        string
	endpoint    func(handler.Handler) http.HandlerFunc
	req         request
	serviceErr  error
	wantStatus  int
	wantMessage string // "" means the message header must be absent
	wantParams  any    // nil means the service must not be called
}

func runAccountCases(t *testing.T, tests []accountCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			accounts := &fakeAccounts{err: tt.serviceErr}
			got := do(t, tt.endpoint(newHandler(accounts, &fakeTransactions{})), tt.req)

			assertResponse(t, got, tt.wantStatus, tt.wantMessage)
			assertParams(t, accounts.got, tt.wantParams)

			if tt.serviceErr != nil || tt.wantParams == nil {
				return
			}

			wantBody := "" // a nil result writes no body
			if accounts.resp != nil {
				wantBody = encoded(t, accounts.resp)
			}

			if got.body != wantBody {
				t.Errorf("body = %s, want %s", got.body, wantBody)
			}
		})
	}
}

func TestAccountHandlers_BodyAndQuery(t *testing.T) {
	t.Parallel()

	created := account.CreateAccountParams{OwnerID: "u1", Type: "USER_WALLET", Currency: ngn}
	create := func(h handler.Handler) http.HandlerFunc { return h.CreateAccount }
	list := func(h handler.Handler) http.HandlerFunc { return h.ListAccounts }

	runAccountCases(t, []accountCase{
		{
			name:        "create",
			endpoint:    create,
			req:         request{method: http.MethodPost, target: accountsPath, body: validAccount},
			wantStatus:  http.StatusCreated,
			wantMessage: "account created",
			wantParams:  created,
		},
		{
			name:       "create rejects invalid body",
			endpoint:   create,
			req:        request{method: http.MethodPost, target: accountsPath, body: `{"owner_id":"u1"}`},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "create maps service error",
			endpoint:   create,
			req:        request{method: http.MethodPost, target: accountsPath, body: validAccount},
			serviceErr: account.ErrAccountExists,
			wantStatus: http.StatusConflict,
			wantParams: created,
		},
		{
			name:     "list",
			endpoint: list,
			req: request{
				method: http.MethodGet,
				target: accountsPath + "?page=2&limit=5&currency=NGN&type=USER_WALLET",
			},
			wantStatus: http.StatusOK,
			wantParams: account.ListAccountsParams{Page: 2, Limit: 5, Currency: ngn, Type: "USER_WALLET"},
		},
		{
			name:       "list rejects limit above 100",
			endpoint:   list,
			req:        request{method: http.MethodGet, target: accountsPath + "?limit=500"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "list maps service error",
			endpoint:   list,
			req:        request{method: http.MethodGet, target: accountsPath},
			serviceErr: errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantParams: account.ListAccountsParams{},
		},
	})
}

func TestAccountHandlers_PathID(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse(testID)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	summary := func(h handler.Handler) http.HandlerFunc { return h.GetAccountStatementSummary }
	reconcile := func(h handler.Handler) http.HandlerFunc { return h.ReconcileAccount }

	runAccountCases(t, []accountCase{
		{
			name:       "details",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.GetAccountDetails },
			req:        request{method: http.MethodGet, target: "/x", pathID: testID},
			wantStatus: http.StatusOK,
			wantParams: id,
		},
		{
			name:       "balance",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.GetAccountBalance },
			req:        request{method: http.MethodGet, target: "/x", pathID: testID},
			wantStatus: http.StatusOK,
			wantParams: id,
		},
		{
			name:       "statement summary",
			endpoint:   summary,
			req:        request{method: http.MethodGet, target: "/x?" + statementRange, pathID: testID},
			wantStatus: http.StatusOK,
			wantParams: account.GetStatementSummaryParams{AccountID: id, StartDate: start, EndDate: end},
		},
		{
			name:       "statement summary requires dates",
			endpoint:   summary,
			req:        request{method: http.MethodGet, target: "/x", pathID: testID},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "statement entries",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.GetAccountStatementEntries },
			req: request{
				method: http.MethodGet,
				target: "/x?" + statementRange + "&limit=7&cursor=" + testID,
				pathID: testID,
			},
			wantStatus: http.StatusOK,
			wantParams: account.GetStatementEntriesParams{
				AccountID: id,
				StartDate: start,
				EndDate:   end,
				Limit:     7,
				Cursor:    uuid.NullUUID{UUID: id, Valid: true},
			},
		},
		{
			name:        "reconcile",
			endpoint:    reconcile,
			req:         request{method: http.MethodPost, target: "/x", pathID: testID},
			wantStatus:  http.StatusOK,
			wantMessage: "account reconciled",
			wantParams:  id,
		},
		{
			name:     "reconcile rejects as_of_timestamp",
			endpoint: reconcile,
			req: request{
				method: http.MethodPost,
				target: "/x",
				pathID: testID,
				body:   `{"as_of_timestamp":"2026-01-01T00:00:00Z"}`,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "snapshot",
			endpoint:    func(h handler.Handler) http.HandlerFunc { return h.CreateAccountSnapshot },
			req:         request{method: http.MethodPost, target: "/x", pathID: testID, body: `{"reason":"eod"}`},
			wantStatus:  http.StatusCreated,
			wantMessage: "account snapshot created",
			wantParams:  account.CreateAccountSnapshotParams{ID: id, Reason: "eod"},
		},
	})
}

func TestAccountHandlers_StatusAndSnapshot(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse(testID)
	cursorAt := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	expected := account.StatusActive
	status := func(h handler.Handler) http.HandlerFunc { return h.ChangeAccountStatus }
	history := func(h handler.Handler) http.HandlerFunc { return h.ListAccountStatusHistory }
	freeze := `{"status":"FULLY_FROZEN","reason":"kyc","actor_id":"admin-1"`

	runAccountCases(t, []accountCase{
		{
			name:       "latest snapshot",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.GetLatestAccountSnapshot },
			req:        request{method: http.MethodGet, target: "/x", pathID: testID},
			wantStatus: http.StatusOK,
			wantParams: id,
		},
		{
			name:        "change status records an admin actor",
			endpoint:    status,
			req:         request{method: http.MethodPatch, target: "/x", pathID: testID, body: freeze + `}`},
			wantStatus:  http.StatusOK,
			wantMessage: "account status changed",
			wantParams: account.StatusChangeParams{
				AccountID: id,
				NewStatus: account.StatusFullyFrozen,
				Reason:    "kyc",
				ActorType: account.ActorTypeAdmin,
				ActorID:   "admin-1",
			},
		},
		{
			name:     "change status passes expected status",
			endpoint: status,
			req: request{
				method: http.MethodPatch,
				target: "/x",
				pathID: testID,
				body:   freeze + `,"expected_status":"ACTIVE"}`,
			},
			wantStatus:  http.StatusOK,
			wantMessage: "account status changed",
			wantParams: account.StatusChangeParams{
				AccountID:      id,
				NewStatus:      account.StatusFullyFrozen,
				ExpectedStatus: &expected,
				Reason:         "kyc",
				ActorType:      account.ActorTypeAdmin,
				ActorID:        "admin-1",
			},
		},
		{
			name:       "change status rejects unknown status",
			endpoint:   status,
			req:        request{method: http.MethodPatch, target: "/x", pathID: testID, body: `{"status":"ASLEEP"}`},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "change status maps conflict",
			endpoint:   status,
			req:        request{method: http.MethodPatch, target: "/x", pathID: testID, body: freeze + `}`},
			serviceErr: account.ErrStatusConflict,
			wantStatus: http.StatusConflict,
			wantParams: account.StatusChangeParams{
				AccountID: id,
				NewStatus: account.StatusFullyFrozen,
				Reason:    "kyc",
				ActorType: account.ActorTypeAdmin,
				ActorID:   "admin-1",
			},
		},
		{
			name:       "status history first page",
			endpoint:   history,
			req:        request{method: http.MethodGet, target: "/x?limit=5", pathID: testID},
			wantStatus: http.StatusOK,
			wantParams: account.ListAccountStatusHistoryParams{AccountID: id, Limit: 5},
		},
		{
			name:     "status history with cursor",
			endpoint: history,
			req: request{
				method: http.MethodGet,
				target: "/x?cursor=" + testID + "&cursor_created_at=2026-03-04T05:06:07Z",
				pathID: testID,
			},
			wantStatus: http.StatusOK,
			wantParams: account.ListAccountStatusHistoryParams{
				AccountID:       id,
				CursorID:        uuid.NullUUID{UUID: id, Valid: true},
				CursorCreatedAt: &cursorAt,
			},
		},
		{
			name:       "status history rejects cursor without created_at",
			endpoint:   history,
			req:        request{method: http.MethodGet, target: "/x?cursor=" + testID, pathID: testID},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "status history rejects created_at without cursor",
			endpoint: history,
			req: request{
				method: http.MethodGet,
				target: "/x?cursor_created_at=2026-03-04T05:06:07Z",
				pathID: testID,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	})
}

// TestAccountHandlers_Errors runs every path-ID account endpoint through the
// error matrix, so each handler's decode-failure and service-failure branches
// are pinned, not just the happy path.
func TestAccountHandlers_Errors(t *testing.T) {
	t.Parallel()

	endpoints := []struct {
		name     string
		endpoint func(handler.Handler) http.HandlerFunc
		req      request
	}{
		{
			name:     "details",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.GetAccountDetails },
			req:      request{method: http.MethodGet, target: "/x"},
		},
		{
			name:     "balance",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.GetAccountBalance },
			req:      request{method: http.MethodGet, target: "/x"},
		},
		{
			name:     "summary",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.GetAccountStatementSummary },
			req:      request{method: http.MethodGet, target: "/x?" + statementRange},
		},
		{
			name:     "entries",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.GetAccountStatementEntries },
			req:      request{method: http.MethodGet, target: "/x?" + statementRange},
		},
		{
			name:     "reconcile",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.ReconcileAccount },
			req:      request{method: http.MethodPost, target: "/x"},
		},
		{
			name:     "snapshot",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.CreateAccountSnapshot },
			req:      request{method: http.MethodPost, target: "/x", body: `{"reason":"eod"}`},
		},
		{
			name:     "latest snapshot",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.GetLatestAccountSnapshot },
			req:      request{method: http.MethodGet, target: "/x"},
		},
		{
			name:     "change status",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.ChangeAccountStatus },
			req: request{
				method: http.MethodPatch,
				target: "/x",
				body:   `{"status":"CLOSED","reason":"r","actor_id":"a"}`,
			},
		},
		{
			name:     "status history",
			endpoint: func(h handler.Handler) http.HandlerFunc { return h.ListAccountStatusHistory },
			req:      request{method: http.MethodGet, target: "/x"},
		},
	}

	for _, ep := range endpoints {
		for _, ec := range errorMatrix(account.ErrAccountNotFound) {
			t.Run(ep.name+"/"+ec.name, func(t *testing.T) {
				t.Parallel()

				req := ep.req
				req.pathID = cmp.Or(ec.pathID, testID)

				accounts := &fakeAccounts{err: ec.serviceErr}
				got := do(t, ep.endpoint(newHandler(accounts, &fakeTransactions{})), req)

				assertResponse(t, got, ec.wantStatus, "")

				if called := accounts.got != nil; called != ec.wantCalled {
					t.Errorf("service called = %v, want %v", called, ec.wantCalled)
				}
			})
		}
	}
}

func assertResponse(t *testing.T, got response, wantStatus int, wantMessage string) {
	t.Helper()

	if got.status != wantStatus {
		t.Errorf("status = %d, want %d; body: %s", got.status, wantStatus, got.body)
	}

	wantHeader := wantMessage != ""
	if got.hasMessage != wantHeader || got.message != wantMessage {
		t.Errorf("message header present=%v value=%q, want present=%v value=%q",
			got.hasMessage, got.message, wantHeader, wantMessage)
	}
}

func assertParams(t *testing.T, got, want any) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("service received %#v, want %#v", got, want)
	}
}
