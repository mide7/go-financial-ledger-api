// Package handler implements the HTTP handlers for the ledger API.
package handler

import (
	"context"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/pagination"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
)

// CreateAccount opens an account from the JSON body.
func (h *handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{status: http.StatusCreated, message: "account created"},
		func(ctx context.Context, req dto.CreateAccountDTO) (*account.Account, error) {
			return h.accountService.CreateAccount(ctx, req.ToDomainParams())
		})
}

// GetAccountDetails returns the account named in the path.
func (h *handler) GetAccountDetails(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPath, status: http.StatusOK},
		func(ctx context.Context, req dto.GetAccountDetailsDTO) (*account.Account, error) {
			return h.accountService.GetAccountDetails(ctx, req.ID)
		})
}

// ListAccounts returns a page of accounts filtered by the query string.
func (h *handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromQuery, status: http.StatusOK},
		func(ctx context.Context, req dto.ListAccountsDTO) (pagination.OffsetPaginatedResult[account.Account], error) {
			return h.accountService.ListAccounts(ctx, req.ToDomainParams())
		})
}

// GetAccountBalance returns the balance of an account.
func (h *handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPath, status: http.StatusOK},
		func(ctx context.Context, req dto.GetAccountBalanceDTO) (*account.GetAccountBalanceResult, error) {
			return h.accountService.GetAccountBalance(ctx, req.ID)
		})
}

// GetAccountStatementSummary returns an account's statement summary for the
// start_date and end_date query parameters.
func (h *handler) GetAccountStatementSummary(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathQuery, status: http.StatusOK},
		func(ctx context.Context, req dto.GetAccountStatementSummaryDTO) (*account.StatementSummary, error) {
			return h.accountService.GetAccountStatementSummary(ctx, req.ToDomainParams())
		})
}

// GetAccountStatementEntries returns a cursor page of an account's statement
// entries for the date range in the query string.
func (h *handler) GetAccountStatementEntries(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathQuery, status: http.StatusOK},
		func(
			ctx context.Context,
			req dto.GetAccountStatementEntriesDTO,
		) (pagination.CursorPaginatedResult[account.StatementEntry], error) {
			return h.accountService.GetAccountStatementEntries(ctx, req.ToDomainParams())
		})
}

// ReconcileAccount compares the account's ledger with its latest snapshot.
func (h *handler) ReconcileAccount(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathBody, status: http.StatusOK, message: "account reconciled"},
		func(ctx context.Context, req dto.ReconcileAccountDTO) (*account.ReconciliationResult, error) {
			return h.accountService.ReconcileAccount(ctx, req.ID)
		})
}

// CreateAccountSnapshot checkpoints the account's current balance.
func (h *handler) CreateAccountSnapshot(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathBody, status: http.StatusCreated, message: "account snapshot created"},
		func(ctx context.Context, req dto.CreateAccountSnapshotDTO) (*account.BalanceSnapshot, error) {
			return h.accountService.CreateAccountSnapshot(ctx, req.ToDomainParams())
		})
}

// GetLatestAccountSnapshot returns the account's most recent balance snapshot.
func (h *handler) GetLatestAccountSnapshot(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPath, status: http.StatusOK},
		func(ctx context.Context, req dto.GetLatestAccountSnapshotDTO) (*account.BalanceSnapshot, error) {
			return h.accountService.GetLatestAccountSnapshot(ctx, req.ID)
		})
}

// ChangeAccountStatus moves the account to the status in the JSON body.
func (h *handler) ChangeAccountStatus(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathBody, status: http.StatusOK, message: "account status changed"},
		func(ctx context.Context, req dto.ChangeAccountStatusDTO) (*account.StatusHistory, error) {
			return h.accountService.ChangeAccountStatus(ctx, req.ToDomainParams())
		})
}

// ListAccountStatusHistory returns a cursor page of the account's status changes.
func (h *handler) ListAccountStatusHistory(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathQuery, status: http.StatusOK},
		func(
			ctx context.Context,
			req dto.ListAccountStatusHistoryDTO,
		) (pagination.CursorPaginatedResult[account.StatusHistory], error) {
			return h.accountService.ListAccountStatusHistory(ctx, req.ToDomainParams())
		})
}
