package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	goValidator "github.com/go-playground/validator/v10"
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
	var ve goValidator.ValidationErrors
	if errors.As(err, &ve) {
		formattedErrs := validation.FormatValidationErrors(ve)
		JSON(w, http.StatusUnprocessableEntity, formattedErrs)
		return
	}

	// Handle invalid JSON
	if errors.Is(err, ErrInvalidJSON) {
		JSON(w, http.StatusUnprocessableEntity, "invalid JSON body")
		return
	}

	// domain errors
	switch {
	default:
		// fallback to internal server error
		JSON(w, http.StatusInternalServerError, "internal server error")
	}
}
