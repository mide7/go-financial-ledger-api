package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	goValidator "github.com/go-playground/validator/v10"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middlewares"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/validation"
)

// SetMessage sets a custom response message for the JSONResponse middleware.
func SetMessage(w http.ResponseWriter, msg string) {
	w.Header().Set(middlewares.HeaderResponseMessage, msg)
}

// JSON writes status code and encodes data as JSON for the middleware to intercept.
func JSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// JSONMsg writes status code, sets a custom response message, and encodes data.
func JSONMsg(w http.ResponseWriter, status int, msg string, v any) {
	SetMessage(w, msg)
	JSON(w, status, v)
}

// Error writes the error to the response writer.
func Error(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	// Handle go-playground/validator errors
	if ve, ok := errors.AsType[goValidator.ValidationErrors](err); ok {
		formattedErrs := validation.FormatValidationErrors(ve)
		JSON(w, http.StatusUnprocessableEntity, formattedErrs)
		return
	}

	// Handle invalid JSON
	if errors.Is(err, ErrInvalidJSON) {
		JSON(w, http.StatusUnprocessableEntity, "invalid JSON body")
		return
	}

	// Handle invalid query parameters
	if errors.Is(err, ErrInvalidQuery) {
		JSON(w, http.StatusUnprocessableEntity, "invalid query parameters")
		return
	}

	// Handle invalid path parameters
	if errors.Is(err, ErrInvalidPath) {
		JSON(w, http.StatusUnprocessableEntity, "invalid path parameters")
		return
	}

	// domain errors
	switch {
	// Account errors
	case errors.Is(err, account.ErrAccountNotFound):
		JSON(w, http.StatusNotFound, "account not found")
	case errors.Is(err, account.ErrInvalidCurrency):
		JSON(w, http.StatusUnprocessableEntity, "invalid currency code")
	case errors.Is(err, account.ErrInvalidType):
		JSON(w, http.StatusUnprocessableEntity, "invalid account type")
	case errors.Is(err, account.ErrAccountExists):
		JSON(w, http.StatusConflict, "account already exists")
	case errors.Is(err, account.ErrDiscrepancyFound):
		JSON(w, http.StatusConflict, "account balance discrepancy detected")
	// Transaction errors
	case errors.Is(err, transaction.ErrDuplicateReference):
		JSON(w, http.StatusUnprocessableEntity, "duplicate reference")
	case errors.Is(err, transaction.ErrTransactionNotFound):
		JSON(w, http.StatusNotFound, "transaction not found")
	case errors.Is(err, transaction.ErrInvalidCurrency):
		JSON(w, http.StatusUnprocessableEntity, "invalid currency code")
	case errors.Is(err, transaction.ErrAtLeastTwoEntries):
		JSON(w, http.StatusUnprocessableEntity, "transaction must contain at least two entries")
	case errors.Is(err, transaction.ErrUnbalancedTransaction):
		JSON(w, http.StatusUnprocessableEntity, "debits and credits must balance to zero")
	case errors.Is(err, transaction.ErrInvalidEntryType):
		JSON(w, http.StatusUnprocessableEntity, "invalid entry type")
	// fallback to internal server error
	default:
		JSON(w, http.StatusInternalServerError, "an unexpected error occurred")
	}
}
