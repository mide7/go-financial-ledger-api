package middlewares

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type StandardResponse struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
	Errors    any    `json:"errors"`
}

func JSONResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initialize the wrapped writer with a live buffer
		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default status
			body:           bytes.NewBuffer(nil),
			wroteHeader:    false,
		}

		// Pass the wrapped writer down the chain
		next.ServeHTTP(wrapped, r)

		// If handler opted out, flush the buffer as-is and return
		if r.Context().Value(SkipJSONResponse) == true {
			for key, vals := range wrapped.Header() {
				for _, val := range vals {
					w.Header().Add(key, val)
				}
			}
			w.WriteHeader(wrapped.statusCode)
			w.Write(wrapped.body.Bytes())
			return
		}

		// Parse whatever the inner handler wrote
		var data any
		var errs any
		innerBody := wrapped.body.Bytes()

		if wrapped.statusCode >= 400 {
			// If it's an error status, treat inner body as the errors field
			if len(innerBody) > 0 {
				errs = string(innerBody)
			}
		} else {
			// If it's a success status, try to parse JSON, otherwise treat as raw string/bytes
			if len(innerBody) > 0 {
				if json.Valid(innerBody) {
					data = json.RawMessage(innerBody) // Keeps inner JSON formatted nicely
				} else {
					data = string(innerBody)
				}
			}
		}

		// Build your unified structure
		response := StandardResponse{
			Status:    wrapped.statusCode,
			Message:   http.StatusText(wrapped.statusCode),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data:      data,
			Errors:    errs,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(wrapped.statusCode)
		json.NewEncoder(w).Encode(response)
	})
}
