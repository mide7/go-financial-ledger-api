package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

func TestChain(t *testing.T) {
	t.Parallel()

	var calls []string

	record := func(name string) middleware.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name+" in")

				next.ServeHTTP(w, r)

				calls = append(calls, name+" out")
			})
		}
	}

	inner := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls = append(calls, "handler")
	})
	handler := middleware.Chain(record("first"), record("second"))(inner)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	want := "first in, second in, handler, second out, first out"
	if got := strings.Join(calls, ", "); got != want {
		t.Errorf("call order = %s, want %s", got, want)
	}
}

func TestChain_Empty(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	middleware.Chain()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}
