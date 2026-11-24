package handler

import (
	"context"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
)

// ListUnresolvedDiscrepancies returns every unresolved discrepancy, newest first.
func (h *handler) ListUnresolvedDiscrepancies(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromQuery, status: http.StatusOK},
		func(ctx context.Context, _ struct{}) ([]account.DiscrepancyRecord, error) {
			return h.accountService.ListUnresolvedDiscrepancies(ctx)
		})
}

// ResolveDiscrepancy marks a discrepancy resolved. It answers with a message and no body.
func (h *handler) ResolveDiscrepancy(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathBody, status: http.StatusOK, message: "discrepancy resolved"},
		func(ctx context.Context, req dto.ResolveDiscrepancyDTO) (any, error) {
			return nil, h.accountService.ResolveDiscrepancy(ctx, req.ID, req.Notes)
		})
}

// ReconcileAllAccounts reconciles every account and summarises the run.
func (h *handler) ReconcileAllAccounts(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromQuery, status: http.StatusOK, message: "reconciliation run finished"},
		func(ctx context.Context, _ struct{}) (*account.ReconciliationRun, error) {
			return h.accountService.ReconcileAllAccounts(ctx)
		})
}
