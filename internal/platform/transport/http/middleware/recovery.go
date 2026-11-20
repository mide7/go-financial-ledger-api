package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Recovery returns middleware that turns a handler panic into a 500 response
// and logs the panic value and stack trace to log.
func Recovery(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return recoverer(next, log)
	}
}

func recoverer(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace for debugging
				stack := debug.Stack()

				log.ErrorContext(r.Context(), "panic recovered in request handler",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(stack),
				)

				// Guard against headers already being written
				if w.Header().Get("Content-Type") == "" {
					w.Header().Set("Content-Type", "application/json")
				}

				w.WriteHeader(http.StatusInternalServerError)

				// Match your StandardResponse structure for consistency
				response := StandardResponse{
					Status:    http.StatusInternalServerError,
					Message:   http.StatusText(http.StatusInternalServerError),
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Data:      nil,
					Errors:    "internal server error",
				}

				_ = json.NewEncoder(w).Encode(response)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
