package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

func TestRecovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantBody   bool // a StandardResponse error envelope is expected
		wantLogged bool
	}{
		{
			name:       "panic becomes 500 envelope",
			handler:    func(http.ResponseWriter, *http.Request) { panic("boom") },
			wantStatus: http.StatusInternalServerError,
			wantBody:   true,
			wantLogged: true,
		},
		{
			name:       "normal response passes through",
			handler:    func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer

			rec := httptest.NewRecorder()
			middleware.Recovery(slog.New(slog.NewJSONHandler(&logs, nil)))(tt.handler).
				ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			logged := bytes.Contains(logs.Bytes(), []byte("panic recovered")) && bytes.Contains(logs.Bytes(), []byte("boom"))
			if logged != tt.wantLogged {
				t.Errorf("panic logged = %v, want %v; logs: %s", logged, tt.wantLogged, logs.String())
			}

			if !tt.wantBody {
				return
			}

			var got middleware.StandardResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body %s is not a StandardResponse: %v", rec.Body, err)
			}

			if got.Status != http.StatusInternalServerError || got.Errors != "internal server error" || got.Data != nil {
				t.Errorf("envelope = %+v", got)
			}
		})
	}
}

// TestRecovery_InsideJSONResponse checks the production order (Recovery
// outermost): a panic discards the partly buffered response and the client
// gets the recovery envelope instead.
func TestRecovery_InsideJSONResponse(t *testing.T) {
	t.Parallel()

	stack := middleware.Chain(middleware.Recovery(slog.New(slog.DiscardHandler)), middleware.JSONResponse)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"partial":true}`))

		panic("boom")
	}))

	rec := httptest.NewRecorder()
	stack.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	var got middleware.StandardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusInternalServerError || got.Data != nil {
		t.Errorf("got status %d body %s (err %v); want a clean 500 envelope", rec.Code, rec.Body, err)
	}
}
