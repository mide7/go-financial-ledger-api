package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/validation"
	"github.com/mide7/go-financial-ledger-api/internal/validator"
)

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountDTO

	if err := json.NewDecoder(r.Body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := validator.Struct(&req); err != nil {
		validation.WriteValidationErrors(w, err)
		return
	}

	account, err := h.accountService.CreateAccount(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
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
