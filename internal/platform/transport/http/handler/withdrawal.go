package handler

import (
	"context"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/withdrawal"
	"github.com/mide7/go-financial-ledger-api/internal/pagination"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
)

// CreateWithdrawal locks funds and opens a withdrawal from the JSON body.
func (h *handler) CreateWithdrawal(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{status: http.StatusCreated, message: "withdrawal created"},
		func(ctx context.Context, req dto.CreateWithdrawalDTO) (*withdrawal.Withdrawal, error) {
			return h.withdrawalService.CreateWithdrawal(ctx, req.ToDomainParams())
		})
}

// GetWithdrawal returns the withdrawal named in the path.
func (h *handler) GetWithdrawal(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPath, status: http.StatusOK},
		func(ctx context.Context, req dto.GetWithdrawalDTO) (*withdrawal.Withdrawal, error) {
			return h.withdrawalService.GetWithdrawal(ctx, req.ID)
		})
}

// ListWithdrawals returns a cursor page of withdrawals filtered by the query string.
func (h *handler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromQuery, status: http.StatusOK},
		func(
			ctx context.Context,
			req dto.ListWithdrawalsDTO,
		) (pagination.CursorPaginatedResult[withdrawal.Withdrawal], error) {
			return h.withdrawalService.ListWithdrawals(ctx, req.ToDomainParams())
		})
}

// SettleWithdrawal records that the processor paid the withdrawal out.
func (h *handler) SettleWithdrawal(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathBody, status: http.StatusOK, message: "withdrawal settled"},
		func(ctx context.Context, req dto.SettleWithdrawalDTO) (*withdrawal.Withdrawal, error) {
			return h.withdrawalService.SettleWithdrawal(ctx, req.ToDomainParams())
		})
}

// FailWithdrawal records that the payout failed and releases the locked funds.
func (h *handler) FailWithdrawal(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathBody, status: http.StatusOK, message: "withdrawal failed"},
		func(ctx context.Context, req dto.FailWithdrawalDTO) (*withdrawal.Withdrawal, error) {
			return h.withdrawalService.FailWithdrawal(ctx, req.ToDomainParams())
		})
}
