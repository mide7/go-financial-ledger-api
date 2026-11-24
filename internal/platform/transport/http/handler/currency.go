package handler

import (
	"context"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// ListCurrencies returns the active currencies sorted by code.
func (h *handler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromQuery, status: http.StatusOK},
		func(context.Context, struct{}) ([]currency.Currency, error) {
			return h.currencies.List(), nil
		})
}
