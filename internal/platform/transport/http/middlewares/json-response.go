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

		// Copy headers set by downstream handlers to the underlying ResponseWriter
		for key, vals := range wrapped.Header() {
			for _, val := range vals {
				w.Header().Add(key, val)
			}
		}

		// If handler opted out, flush the buffer as-is and return
		if skip, ok := r.Context().Value(SkipJSONResponse).(bool); ok && skip {
			w.WriteHeader(wrapped.statusCode)
			w.Write(wrapped.body.Bytes())
			return
		}

		// Extract custom message if set by handler, otherwise default to http.StatusText
		message := wrapped.Header().Get(HeaderResponseMessage)
		w.Header().Del(HeaderResponseMessage)
		if message == "" {
			message = http.StatusText(wrapped.statusCode)
		}

		// Safely process inner body without double-encoding
		var parsedPayload any
		innerBody := bytes.TrimSpace(wrapped.body.Bytes())

		if len(innerBody) > 0 {
			if json.Valid(innerBody) {
				// Keeps inner JSON structure (objects, arrays, strings) intact
				parsedPayload = json.RawMessage(innerBody)
			} else {
				// Fallback for plain-text response bodies
				parsedPayload = string(innerBody)
			}
		}

		// 3. Assign payload to Data (2xx/3xx) or Errors (4xx/5xx)
		var data, errs any
		if wrapped.statusCode >= 400 {
			errs = parsedPayload
		} else {
			data = parsedPayload
		}

		// Build your unified structure
		response := StandardResponse{
			Status:    wrapped.statusCode,
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data:      data,
			Errors:    errs,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(wrapped.statusCode)
		json.NewEncoder(w).Encode(response)
	})
}
