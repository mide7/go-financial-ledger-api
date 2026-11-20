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

func TestLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		wantLevel string
	}{
		{name: "success logs at info", status: http.StatusOK, wantLevel: "INFO"},
		{name: "client error logs at warn", status: http.StatusNotFound, wantLevel: "WARN"},
		{name: "server error logs at error", status: http.StatusInternalServerError, wantLevel: "ERROR"},
		{name: "unimplemented logs at error", status: http.StatusNotImplemented, wantLevel: "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer

			handler := middleware.Logging(slog.New(slog.NewJSONHandler(&logs, nil)))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts?page=2", nil))

			if rec.Code != tt.status {
				t.Errorf("response status = %d, want %d (middleware must not change it)", rec.Code, tt.status)
			}

			var record struct {
				Level    string  `json:"level"`
				Msg      string  `json:"msg"`
				Method   string  `json:"method"`
				Path     string  `json:"path"`
				Status   int     `json:"status"`
				Duration float64 `json:"duration"`
			}
			if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
				t.Fatalf("want exactly one JSON record, got %q: %v", logs.String(), err)
			}

			if record.Level != tt.wantLevel || record.Msg != "http request" || record.Method != http.MethodGet ||
				record.Path != "/api/v1/accounts" || record.Status != tt.status || record.Duration < 0 {
				t.Errorf("record = %+v", record)
			}
		})
	}
}

func TestLogging_DefaultStatusIsOK(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	// A handler that writes a body without calling WriteHeader sends 200.
	handler := middleware.Logging(slog.New(slog.NewJSONHandler(&logs, nil)))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if !bytes.Contains(logs.Bytes(), []byte(`"status":200`)) {
		t.Errorf("logs = %s, want status 200", logs.String())
	}
}
