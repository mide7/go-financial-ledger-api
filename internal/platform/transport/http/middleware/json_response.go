// Package middleware provides the HTTP middleware chain: recovery, logging and response wrapping.
package middleware

import (
	"bytes"
	"cmp"
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
		}

		next.ServeHTTP(wrapped, r)

		for key, vals := range wrapped.Header() {
			for _, val := range vals {
				w.Header().Add(key, val)
			}
		}

		if skip, _ := r.Context().Value(SkipJSONResponse).(bool); skip {
			w.WriteHeader(wrapped.statusCode)
			_, _ = w.Write(wrapped.body.Bytes()) // client may have gone away; nothing useful to do

			return
		}

		message := cmp.Or(wrapped.Header().Get(HeaderResponseMessage), http.StatusText(wrapped.statusCode))
		w.Header().Del(HeaderResponseMessage)

		var payload any

		switch body := bytes.TrimSpace(wrapped.body.Bytes()); {
		case len(body) == 0:
		case json.Valid(body):
			payload = json.RawMessage(body)
		default:
			payload = string(body)
		}

		response := StandardResponse{
			Status:    wrapped.statusCode,
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		if wrapped.statusCode >= http.StatusBadRequest {
			response.Errors = payload
		} else {
			response.Data = payload
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(wrapped.statusCode)
		_ = json.NewEncoder(w).Encode(response) // headers already sent; cannot report a failure
	})
}
