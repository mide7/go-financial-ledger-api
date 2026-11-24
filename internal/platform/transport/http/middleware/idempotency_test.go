package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mide7/go-financial-ledger-api/internal/platform/idempotency"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

// memStore is an in-memory idempotency.Store with the Postgres store's claim
// rules: a new key is claimed, a FAILED key is reclaimed by the same request,
// anything else is returned for the middleware to decide.
type memStore struct {
	mu       sync.Mutex
	records  map[string]*idempotency.Record
	claimErr error
}

func newMemStore() *memStore { return &memStore{records: map[string]*idempotency.Record{}} }

func (s *memStore) Claim(
	_ context.Context,
	key, hash string,
	ttl time.Duration,
) (*idempotency.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.claimErr != nil {
		return nil, false, s.claimErr
	}

	rec, ok := s.records[key]
	if !ok || (rec.Status == idempotency.StatusFailed && rec.RequestHash == hash) {
		s.records[key] = &idempotency.Record{
			Key: key, RequestHash: hash, Status: idempotency.StatusInProgress, ExpiresAt: time.Now().Add(ttl),
		}

		return nil, true, nil
	}

	clone := *rec

	return &clone, false, nil
}

func (s *memStore) Complete(_ context.Context, key string, code int, body []byte) error {
	return s.finish(key, idempotency.StatusCompleted, code, body)
}

func (s *memStore) Fail(_ context.Context, key string, code int) error {
	return s.finish(key, idempotency.StatusFailed, code, nil)
}

func (s *memStore) finish(key, status string, code int, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.records[key]
	if !ok || rec.Status != idempotency.StatusInProgress {
		return idempotency.ErrNotClaimed
	}

	rec.Status, rec.StatusCode, rec.ResponseBody = status, code, body

	return nil
}

func (*memStore) DeleteExpired(context.Context) (int64, error) { return 0, nil }

func (s *memStore) status(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rec, ok := s.records[key]; ok {
		return rec.Status
	}

	return ""
}

// countingHandler answers 201 with a custom message and echoes the body, and
// counts its calls, so tests can tell a replay from a re-run.
type countingHandler struct {
	mu     sync.Mutex
	calls  int
	status int
}

func (h *countingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.calls++
	h.mu.Unlock()

	body, _ := io.ReadAll(r.Body)

	w.Header().Set(middleware.HeaderResponseMessage, "transaction created")
	w.WriteHeader(h.status)
	_ = json.NewEncoder(w).Encode(map[string]string{"echo": string(body)})
}

func (h *countingHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.calls
}

// serveIdem sends one POST through JSONResponse → Idempotency → next, as the
// server stacks them, and returns the recorder.
func serveIdem(t *testing.T, store idempotency.Store, next http.Handler, key *string, body string) *httptest.ResponseRecorder {
	t.Helper()

	stack := middleware.Chain(
		middleware.JSONResponse,
		middleware.Idempotency(store, time.Hour, slog.New(slog.DiscardHandler)),
	)(next)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/transactions", strings.NewReader(body))
	if key != nil {
		req.Header.Set(middleware.HeaderIdempotencyKey, *key)
	}

	rec := httptest.NewRecorder()
	stack.ServeHTTP(rec, req)

	return rec
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) middleware.StandardResponse {
	t.Helper()

	var env middleware.StandardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope %q: %v", rec.Body.String(), err)
	}

	return env
}

func TestIdempotency_ReplaysCompletedResponse(t *testing.T) {
	t.Parallel()

	store, next := newMemStore(), &countingHandler{status: http.StatusCreated}

	first := serveIdem(t, store, next, new("k1"), `{"amount":100}`)
	second := serveIdem(t, store, next, new("k1"), `{"amount":100}`)

	if next.count() != 1 {
		t.Fatalf("handler ran %d times, want 1", next.count())
	}

	if second.Code != http.StatusCreated || second.Header().Get(middleware.HeaderIdempotentReplayed) != "true" {
		t.Errorf("replay status = %d, replayed header = %q; want 201, true",
			second.Code, second.Header().Get(middleware.HeaderIdempotentReplayed))
	}

	if first.Header().Get(middleware.HeaderIdempotentReplayed) != "" {
		t.Error("first response is marked as replayed")
	}

	a, b := decodeEnvelope(t, first), decodeEnvelope(t, second)
	if a.Message != "transaction created" || b.Message != a.Message {
		t.Errorf("messages = %q, %q; want both %q", a.Message, b.Message, "transaction created")
	}

	if da, db := mustJSON(t, a.Data), mustJSON(t, b.Data); da != db {
		t.Errorf("replayed data = %s, want %s", db, da)
	}
}

func TestIdempotency_Rejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*memStore)
		key      *string
		body     string
		wantCode int
	}{
		{name: "empty key", key: new(""), wantCode: http.StatusBadRequest},
		{name: "key too long", key: new(strings.Repeat("k", middleware.MaxIdempotencyKeyLength+1)), wantCode: http.StatusBadRequest},
		{
			name:     "body too large",
			key:      new("big"),
			body:     strings.Repeat("x", 1<<20+1),
			wantCode: http.StatusRequestEntityTooLarge,
		},
		{
			name: "key reused for a different request",
			setup: func(s *memStore) {
				s.records["k"] = &idempotency.Record{Key: "k", RequestHash: "other", Status: idempotency.StatusCompleted}
			},
			key:      new("k"),
			body:     `{}`,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "store failure",
			setup:    func(s *memStore) { s.claimErr = errors.New("db down") },
			key:      new("k"),
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store, next := newMemStore(), &countingHandler{status: http.StatusCreated}
			if tt.setup != nil {
				tt.setup(store)
			}

			rec := serveIdem(t, store, next, tt.key, tt.body)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d (body %s)", rec.Code, tt.wantCode, rec.Body)
			}

			if next.count() != 0 {
				t.Errorf("handler ran %d times, want 0", next.count())
			}
		})
	}
}

func TestIdempotency_InProgress(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	release, entered := make(chan struct{}), make(chan struct{})

	slow := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusCreated)
	})

	done := make(chan *httptest.ResponseRecorder)

	go func() { done <- serveIdem(t, store, slow, new("k"), `{}`) }()

	<-entered

	if rec := serveIdem(t, store, slow, new("k"), `{}`); rec.Code != http.StatusConflict {
		t.Errorf("concurrent retry status = %d, want 409", rec.Code)
	}

	close(release)

	if rec := <-done; rec.Code != http.StatusCreated {
		t.Errorf("first request status = %d, want 201", rec.Code)
	}
}

func TestIdempotency_ServerErrorAllowsRetry(t *testing.T) {
	t.Parallel()

	store, next := newMemStore(), &countingHandler{status: http.StatusInternalServerError}

	if rec := serveIdem(t, store, next, new("k"), `{}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("first status = %d, want 500", rec.Code)
	}

	if got := store.status("k"); got != idempotency.StatusFailed {
		t.Fatalf("record status = %q, want FAILED", got)
	}

	next.status = http.StatusCreated

	if rec := serveIdem(t, store, next, new("k"), `{}`); rec.Code != http.StatusCreated {
		t.Errorf("retry status = %d, want 201", rec.Code)
	}

	if next.count() != 2 || store.status("k") != idempotency.StatusCompleted {
		t.Errorf("handler ran %d times, record %q; want 2, COMPLETED", next.count(), store.status("k"))
	}
}

func TestIdempotency_ClientErrorIsReplayed(t *testing.T) {
	t.Parallel()

	store, next := newMemStore(), &countingHandler{status: http.StatusUnprocessableEntity}

	serveIdem(t, store, next, new("k"), `{}`)
	rec := serveIdem(t, store, next, new("k"), `{}`)

	if rec.Code != http.StatusUnprocessableEntity || next.count() != 1 {
		t.Errorf("retry status = %d after %d runs; want the saved 422 after 1", rec.Code, next.count())
	}
}

func TestIdempotency_PanicMarksFailed(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })

	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic was swallowed; want it re-raised for Recovery")
			}
		}()

		serveIdem(t, store, boom, new("k"), `{}`)
	}()

	if got := store.status("k"); got != idempotency.StatusFailed {
		t.Errorf("record status = %q, want FAILED", got)
	}
}

func TestIdempotency_NoHeaderPassesThrough(t *testing.T) {
	t.Parallel()

	store, next := newMemStore(), &countingHandler{status: http.StatusCreated}

	serveIdem(t, store, next, nil, `{}`)
	serveIdem(t, store, next, nil, `{}`)

	if next.count() != 2 || len(store.records) != 0 {
		t.Errorf("handler ran %d times with %d records; want 2 runs, no records", next.count(), len(store.records))
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return string(b)
}
