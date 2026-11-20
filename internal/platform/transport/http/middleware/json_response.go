// Package middleware provides the HTTP middleware chain: recovery, logging and response wrapping.
package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// StandardResponse is the envelope every JSON response is wrapped in.
type StandardResponse struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
	Errors    any    `json:"errors"`
}

// JSONResponse buffers the handler's output and wraps it in a StandardResponse.
// Bodies with a status of 400 or above go in Errors, others in Data.
func JSONResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           bytes.NewBuffer(nil),
			wroteHeader:    false,
		}

		next.ServeHTTP(wrapped, r)

		for key, vals := range wrapped.Header() {
			for _, val := range vals {
				w.Header().Add(key, val)
			}
		}

		if skip, ok := r.Context().Value(SkipJSONResponse).(bool); ok && skip {
			w.WriteHeader(wrapped.statusCode)
			_, _ = w.Write(wrapped.body.Bytes()) // client may have gone away; nothing useful to do

			return
		}

		message := wrapped.Header().Get(HeaderResponseMessage)
		w.Header().Del(HeaderResponseMessage)

		if message == "" {
			message = http.StatusText(wrapped.statusCode)
		}

		var parsedPayload any

		innerBody := bytes.TrimSpace(wrapped.body.Bytes())

		if len(innerBody) > 0 {
			if json.Valid(innerBody) {
				parsedPayload = json.RawMessage(innerBody)
			} else {
				parsedPayload = string(innerBody)
			}
		}

		var data, errs any
		if wrapped.statusCode >= 400 {
			errs = parsedPayload
		} else {
			data = parsedPayload
		}

		response := StandardResponse{
			Status:    wrapped.statusCode,
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data:      data,
			Errors:    errs,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(wrapped.statusCode)
		_ = json.NewEncoder(w).Encode(response) // headers already sent; cannot report a failure
	})
}
