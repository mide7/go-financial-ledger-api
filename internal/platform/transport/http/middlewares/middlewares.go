package middlewares

import (
	"bytes"
	"net/http"
	"slices"
)

type contextKey string

const SkipJSONResponse contextKey = "skipJSONResponse"
const HeaderResponseMessage string = "X-Response-Message"

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

func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for _, x := range slices.Backward(middlewares) {
			next = x(next)
		}

		return next
	}
}
