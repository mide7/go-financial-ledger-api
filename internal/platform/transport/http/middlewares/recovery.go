package middlewares

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace for debugging
				stack := debug.Stack()

				slog.Error("panic recovered in request handler",
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
