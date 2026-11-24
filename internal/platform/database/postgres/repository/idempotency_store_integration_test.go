//go:build integration

package repository_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/repository"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
	"github.com/mide7/go-financial-ledger-api/internal/platform/idempotency"
)

func newIdempotencyStore(t *testing.T) idempotency.Store {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	return repository.NewIdempotencyStore(sqlc.New(pool))
}

// TestIdempotencyStore_Lifecycle walks one key through claim, retry while in
// progress, completion and replay.
func TestIdempotencyStore_Lifecycle(t *testing.T) {
	t.Parallel()

	store := newIdempotencyStore(t)
	ctx := t.Context()
	key := "it-" + uuid.NewString()

	mustClaim(t, store, key, "hash-a", time.Hour, true)

	existing := mustClaim(t, store, key, "hash-a", time.Hour, false)
	if existing.Status != idempotency.StatusInProgress {
		t.Errorf("status while running = %q, want IN_PROGRESS", existing.Status)
	}

	body := []byte(`{"message":"transaction created","body":{"id":"x"}}`)
	if err := store.Complete(ctx, key, http.StatusCreated, body); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if err := store.Complete(ctx, key, http.StatusCreated, body); !errors.Is(err, idempotency.ErrNotClaimed) {
		t.Errorf("second Complete error = %v, want ErrNotClaimed", err)
	}

	replayed := mustClaim(t, store, key, "hash-a", time.Hour, false)
	if replayed.Status != idempotency.StatusCompleted || replayed.StatusCode != http.StatusCreated || !sameJSON(t, replayed.ResponseBody, body) {
		t.Errorf("replayed record = %+v, want COMPLETED 201 with the saved body", replayed)
	}

	if other := mustClaim(t, store, key, "hash-b", time.Hour, false); other.RequestHash != "hash-a" {
		t.Errorf("different request got record with hash %q, want the original hash-a", other.RequestHash)
	}
}

// TestIdempotencyStore_Reclaim checks that FAILED keys are reclaimable only by
// the same request, and expired keys by any request.
func TestIdempotencyStore_Reclaim(t *testing.T) {
	t.Parallel()

	store := newIdempotencyStore(t)
	ctx := t.Context()

	failed := "it-" + uuid.NewString()
	mustClaim(t, store, failed, "hash-a", time.Hour, true)

	if err := store.Fail(ctx, failed, http.StatusInternalServerError); err != nil {
		t.Fatalf("Fail: %v", err)
	}

	mustClaim(t, store, failed, "hash-b", time.Hour, false)
	mustClaim(t, store, failed, "hash-a", time.Hour, true)

	expired := "it-" + uuid.NewString()
	mustClaim(t, store, expired, "hash-a", -time.Minute, true)
	mustClaim(t, store, expired, "hash-b", time.Hour, true)
}

func TestIdempotencyStore_DeleteExpired(t *testing.T) {
	t.Parallel()

	store := newIdempotencyStore(t)
	key := "it-" + uuid.NewString()

	mustClaim(t, store, key, "hash-a", -time.Minute, true)

	n, err := store.DeleteExpired(t.Context())
	if err != nil || n < 1 {
		t.Fatalf("DeleteExpired = %d, %v; want at least 1", n, err)
	}

	// Gone, so the next claim is a fresh insert, even for another request.
	mustClaim(t, store, key, "hash-b", time.Hour, true)
}

// TestIdempotencyStore_ConcurrentClaims races several claims for one key:
// exactly one wins.
func TestIdempotencyStore_ConcurrentClaims(t *testing.T) {
	t.Parallel()

	store := newIdempotencyStore(t)
	key := "it-" + uuid.NewString()

	const racers = 8

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		wins int
	)

	for range racers {
		wg.Go(func() {
			_, claimed, err := store.Claim(t.Context(), key, "hash-a", time.Hour)
			if err != nil {
				t.Errorf("Claim: %v", err)
				return
			}

			if claimed {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	if wins != 1 {
		t.Errorf("%d claims won, want exactly 1", wins)
	}
}

func mustClaim(
	t *testing.T,
	store idempotency.Store,
	key, hash string,
	ttl time.Duration,
	wantClaimed bool,
) *idempotency.Record {
	t.Helper()

	existing, claimed, err := store.Claim(t.Context(), key, hash, ttl)
	if err != nil {
		t.Fatalf("Claim(%s, %s): %v", key, hash, err)
	}

	if claimed != wantClaimed {
		t.Fatalf("Claim(%s, %s) claimed = %v, want %v (existing %+v)", key, hash, claimed, wantClaimed, existing)
	}

	return existing
}

// sameJSON compares JSON documents by value; JSONB doesn't keep key order or spacing.
func sameJSON(t *testing.T, a, b []byte) bool {
	t.Helper()

	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("unmarshal %s: %v", a, err)
	}

	if err := json.Unmarshal(b, &vb); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}

	ja, _ := json.Marshal(va)
	jb, _ := json.Marshal(vb)

	return string(ja) == string(jb)
}
