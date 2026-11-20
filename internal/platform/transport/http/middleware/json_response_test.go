package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

// jsonNull is how an absent data or errors payload is encoded.
const jsonNull = "null"

// envelope mirrors StandardResponse with raw payloads, so tests can tell a
// JSON null apart from a missing or re-encoded value.
type envelope struct {
	Status    int             `json:"status"`
	Message   string          `json:"message"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
	Errors    json.RawMessage `json:"errors"`
}

func serveJSON(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	middleware.JSONResponse(handler).ServeHTTP(
		rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil),
	)

	return rec
}

func TestJSONResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantStatus  int
		wantMessage string
		wantData    string
		wantErrors  string
	}{
		{
			name: "success body goes in data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"a1"}`))
			},
			wantStatus: http.StatusCreated, wantMessage: "Created", wantData: `{"id":"a1"}`, wantErrors: jsonNull,
		},
		{
			name: "error body goes in errors",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`"account not found"`))
			},
			wantStatus: http.StatusNotFound, wantMessage: "Not Found",
			wantData: jsonNull, wantErrors: `"account not found"`,
		},
		{
			name: "custom message header replaces status text",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(middleware.HeaderResponseMessage, "account created")
				w.WriteHeader(http.StatusCreated)
			},
			wantStatus: http.StatusCreated, wantMessage: "account created", wantData: jsonNull, wantErrors: jsonNull,
		},
		{
			name: "plain text body becomes a JSON string",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("hello"))
			},
			wantStatus: http.StatusOK, wantMessage: "OK", wantData: `"hello"`, wantErrors: jsonNull,
		},
		{
			name: "first WriteHeader wins",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: http.StatusAccepted, wantMessage: "Accepted", wantData: jsonNull, wantErrors: jsonNull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := serveJSON(t, tt.handler)

			if rec.Code != tt.wantStatus {
				t.Errorf("HTTP status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			if h := rec.Header().Get(middleware.HeaderResponseMessage); h != "" {
				t.Errorf("internal message header leaked to client: %q", h)
			}

			var env envelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("body %s is not an envelope: %v", rec.Body, err)
			}

			if env.Status != tt.wantStatus || env.Message != tt.wantMessage ||
				string(env.Data) != tt.wantData || string(env.Errors) != tt.wantErrors {
				t.Errorf("envelope = status %d, message %q, data %s, errors %s; want %d, %q, %s, %s",
					env.Status, env.Message, env.Data, env.Errors,
					tt.wantStatus, tt.wantMessage, tt.wantData, tt.wantErrors)
			}

			if _, err := time.Parse(time.RFC3339, env.Timestamp); err != nil {
				t.Errorf("timestamp %q is not RFC 3339: %v", env.Timestamp, err)
			}
		})
	}
}

func TestJSONResponse_CopiesHandlerHeaders(t *testing.T) {
	t.Parallel()

	rec := serveJSON(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "r-1")
	})

	if got := rec.Header().Get("X-Request-Id"); got != "r-1" {
		t.Errorf("X-Request-Id = %q, want r-1", got)
	}
}

// TestJSONResponse_Skip covers the opt-out. The flag is read from the request
// context before the handler runs, so it must be set by an outer middleware;
// a handler cannot opt out by deriving a new context.
func TestJSONResponse_Skip(t *testing.T) {
	t.Parallel()

	inner := middleware.JSONResponse(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("raw bytes"))
	}))

	skip := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inner.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), middleware.SkipJSONResponse, true)))
	})

	rec := httptest.NewRecorder()
	skip.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rec.Code != http.StatusAccepted || rec.Body.String() != "raw bytes" {
		t.Errorf("got status %d body %q, want 202 and the unwrapped body", rec.Code, rec.Body)
	}
}
