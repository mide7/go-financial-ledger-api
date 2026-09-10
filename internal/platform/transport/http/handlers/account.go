package handlers

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountDTO

	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	account, err := h.accountService.CreateAccount(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSONMsg(w, http.StatusCreated, "account created", account)
}

func (h *Handler) GetAccountDetails(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
}

// GetAccountBalance returns the balance of an account
func (h *Handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
}

// GetAccountStatement returns the statement of an account
func (h *Handler) GetAccountStatement(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
}

func (h *Handler) GetAccountEntries(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
}
