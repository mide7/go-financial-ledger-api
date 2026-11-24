package middleware

import (
	"bytes"
	"net/http"
	"slices"
)

type contextKey string

// SkipJSONResponse is a context key; set it to true to send the body unwrapped.
// HeaderResponseMessage carries a handler's custom message to JSONResponse.
const (
	SkipJSONResponse      contextKey = "skipJSONResponse"
	HeaderResponseMessage string     = "X-Response-Message"
)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

type wrappedWriter struct {
	http.ResponseWriter
	statusCode  int
	body        *bytes.Buffer
	wroteHeader bool
}

func (w *wrappedWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}

	w.wroteHeader = true
	w.statusCode = code
}

func (w *wrappedWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

// Chain composes middlewares so the first one listed runs outermost.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for _, x := range slices.Backward(middlewares) {
			next = x(next)
		}

		return next
	}
}
