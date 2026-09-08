package middlewares

import (
	"log/slog"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sw := &statusWriter{w, http.StatusOK}

		next.ServeHTTP(sw, r)

		slog.Info("Incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}
