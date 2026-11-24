package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
)

type pinger struct{ err error }

func (p pinger) Ping(context.Context) error { return p.err }

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantDB     bool
	}{
		{name: "database reachable", wantStatus: http.StatusOK, wantDB: true},
		{name: "database down", pingErr: errors.New("dial tcp: refused"), wantStatus: http.StatusServiceUnavailable, wantDB: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			handler.NewHealthCheckHandler(pinger{err: tt.pingErr}).
				HealthCheck(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health/", nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			var body struct {
				DB bool `json:"db"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.DB != tt.wantDB {
				t.Errorf("body = %s (err %v), want db=%v", rec.Body, err, tt.wantDB)
			}
		})
	}
}
