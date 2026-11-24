package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/mide7/go-financial-ledger-api/internal/platform/idempotency"
)

// Idempotency header names and limits.
const (
	// HeaderIdempotencyKey is the request header that makes a POST safe to retry.
	HeaderIdempotencyKey = "Idempotency-Key"
	// HeaderIdempotentReplayed is set to "true" on a replayed response.
	HeaderIdempotentReplayed = "Idempotent-Replayed"
	// MaxIdempotencyKeyLength matches idempotency_records.key VARCHAR(256).
	MaxIdempotencyKeyLength = 256
	// maxIdempotentBodyBytes bounds the request body read for fingerprinting.
	maxIdempotentBodyBytes = 1 << 20
)

// storedResponse is what the middleware saves as a record's response_body: the
// handler's raw output and its custom message, both replayed verbatim.
type storedResponse struct {
	Message string          `json:"message,omitempty"`
	Body    json.RawMessage `json:"body"`
}

// Idempotency makes the wrapped handler safe to retry with an Idempotency-Key
// header. Requests without the header pass straight through. With it:
//
//   - the request is fingerprinted by SHA-256 of method, path and body;
//   - the first request claims the key and runs; a response below 500 is saved
//     for ttl and replayed (with Idempotent-Replayed: true) to every retry with
//     the same key and fingerprint;
//   - a 5xx response or a panic marks the key FAILED, so the retry runs again;
//   - a retry while the first is running gets 409, and reusing the key for a
//     different request gets 422; an empty or over-long key is 400, and a body
//     over 1 MiB is 413.
//
// It must run inside JSONResponse, so it saves and replays the handler's
// unwrapped output. Store errors are logged and answered with 500.
func Idempotency(store idempotency.Store, ttl time.Duration, log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			values, present := r.Header[http.CanonicalHeaderKey(HeaderIdempotencyKey)]
			if !present {
				next.ServeHTTP(w, r)
				return
			}

			key := values[0]
			if key == "" || len(key) > MaxIdempotencyKeyLength {
				writeError(w, http.StatusBadRequest, "Idempotency-Key must be 1-256 characters")
				return
			}

			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxIdempotentBodyBytes))
			if err != nil {
				if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
					writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
					return
				}

				writeError(w, http.StatusBadRequest, "could not read request body")

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			hash := fingerprint(r.Method, r.URL.Path, body)

			existing, claimed, err := store.Claim(r.Context(), key, hash, ttl)
			if err != nil {
				log.ErrorContext(r.Context(), "idempotency claim failed", "err", err)
				writeError(w, http.StatusInternalServerError, "an unexpected error occurred")

				return
			}

			if !claimed {
				answerExisting(r.Context(), w, existing, hash, log)
				return
			}

			runAndRecord(w, r, next, store, key, log)
		})
	}
}

// answerExisting replays a completed record, or rejects the retry.
func answerExisting(ctx context.Context, w http.ResponseWriter, rec *idempotency.Record, hash string, log *slog.Logger) {
	switch {
	case rec.RequestHash != hash:
		writeError(w, http.StatusUnprocessableEntity, idempotency.ErrKeyReused.Error())
	case rec.Status == idempotency.StatusCompleted:
		replay(ctx, w, rec, log)
	default: // IN_PROGRESS; a FAILED record for this hash would have been reclaimed
		writeError(w, http.StatusConflict, idempotency.ErrInProgress.Error())
	}
}

// runAndRecord runs next for a freshly claimed key and saves the outcome. The
// save uses a context that survives the client disconnecting.
func runAndRecord(
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	store idempotency.Store,
	key string,
	log *slog.Logger,
) {
	saveCtx := context.WithoutCancel(r.Context())

	defer func() {
		if p := recover(); p != nil {
			if err := store.Fail(saveCtx, key, http.StatusInternalServerError); err != nil {
				log.ErrorContext(saveCtx, "idempotency fail after panic", "err", err)
			}

			panic(p) // let Recovery answer and log it
		}
	}()

	captured := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK, body: bytes.NewBuffer(nil)}
	next.ServeHTTP(captured, r)

	if captured.statusCode >= http.StatusInternalServerError {
		if err := store.Fail(saveCtx, key, captured.statusCode); err != nil {
			log.ErrorContext(saveCtx, "idempotency fail", "err", err)
		}
	} else if err := saveResponse(saveCtx, store, key, w.Header().Get(HeaderResponseMessage), captured); err != nil {
		log.ErrorContext(saveCtx, "idempotency complete", "err", err)
	}

	w.WriteHeader(captured.statusCode)
	_, _ = w.Write(captured.body.Bytes()) // client may have gone away; the record is saved
}

func saveResponse(ctx context.Context, store idempotency.Store, key, message string, captured *wrappedWriter) error {
	doc, err := json.Marshal(storedResponse{Message: message, Body: asJSON(captured.body.Bytes())})
	if err != nil {
		return err
	}

	return store.Complete(ctx, key, captured.statusCode, doc)
}

// replay writes a completed record's saved status, message and body.
func replay(ctx context.Context, w http.ResponseWriter, rec *idempotency.Record, log *slog.Logger) {
	var saved storedResponse
	if err := json.Unmarshal(rec.ResponseBody, &saved); err != nil {
		log.ErrorContext(ctx, "idempotency replay: bad stored response", "key", rec.Key, "err", err)
		writeError(w, http.StatusInternalServerError, "an unexpected error occurred")

		return
	}

	if saved.Message != "" {
		w.Header().Set(HeaderResponseMessage, saved.Message)
	}

	w.Header().Set(HeaderIdempotentReplayed, "true")
	w.WriteHeader(rec.StatusCode)

	if !bytes.Equal(saved.Body, []byte("null")) {
		_, _ = w.Write(saved.Body)
	}
}

// asJSON returns body as a JSON value for storage: itself when it's valid JSON,
// null when empty, and a JSON string otherwise.
func asJSON(body []byte) json.RawMessage {
	body = bytes.TrimSpace(body)

	switch {
	case len(body) == 0:
		return json.RawMessage("null")
	case json.Valid(body):
		return body
	default:
		quoted, _ := json.Marshal(string(body)) // marshalling a string can't fail
		return quoted
	}
}

// fingerprint identifies a request for an idempotency key: SHA-256, in hex, of
// the method, path and body.
func fingerprint(method, path string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(method + "\n" + path + "\n"))
	h.Write(body)

	return hex.EncodeToString(h.Sum(nil))
}

// writeError writes a JSON string body, as httputil.Error does, for
// JSONResponse to wrap. (httputil imports this package, so it can't be used here.)
func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(message)
}
