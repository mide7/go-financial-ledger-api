package handler_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
)

func TestDiscrepancyHandlers(t *testing.T) {
	t.Parallel()

	resolve := func(h handler.Handler) http.HandlerFunc { return h.ResolveDiscrepancy }
	resolveReq := request{method: http.MethodPost, target: "/x", pathID: testID, body: `{"notes":"fee posted twice"}`}
	resolved := resolveCall{ID: uuid.MustParse(testID), Notes: "fee posted twice"}

	runAccountCases(t, []accountCase{
		{
			name:       "list unresolved",
			endpoint:   func(h handler.Handler) http.HandlerFunc { return h.ListUnresolvedDiscrepancies },
			req:        request{method: http.MethodGet, target: "/discrepancies"},
			wantStatus: http.StatusOK,
			wantParams: struct{}{},
		},
		{
			name:        "resolve",
			endpoint:    resolve,
			req:         resolveReq,
			wantStatus:  http.StatusOK,
			wantMessage: "discrepancy resolved",
			wantParams:  resolved,
		},
		{
			name:       "resolve requires notes",
			endpoint:   resolve,
			req:        request{method: http.MethodPost, target: "/x", pathID: testID, body: `{}`},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "resolve rejects malformed ID",
			endpoint:   resolve,
			req:        request{method: http.MethodPost, target: "/x", pathID: "nope", body: `{"notes":"n"}`},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "resolve maps not found",
			endpoint:   resolve,
			req:        resolveReq,
			serviceErr: account.ErrDiscrepancyNotFound,
			wantStatus: http.StatusNotFound,
			wantParams: resolved,
		},
	})
}

func TestListCurrencies(t *testing.T) {
	t.Parallel()

	list := []currency.Currency{
		{Code: currency.EUR, Name: "Euro", Exponent: 2, IsActive: true},
		{Code: currency.NGN, Name: "Naira", Exponent: 2, IsActive: true},
	}
	h := handler.NewHandler(&fakeAccounts{}, &fakeTransactions{}, fakeCurrencies{list: list}, slog.New(slog.DiscardHandler))

	got := do(t, h.ListCurrencies, request{method: http.MethodGet, target: "/currencies"})

	assertResponse(t, got, http.StatusOK, "")

	if want := encoded(t, list); got.body != want {
		t.Errorf("body = %s, want %s", got.body, want)
	}
}
