package httputil_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

const testID = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"

type decodeTarget struct {
	ID     uuid.UUID     `form:"id"`
	Since  time.Time     `form:"since"`
	Cursor uuid.NullUUID `form:"cursor"`
	Limit  int           `form:"limit"`
	Name   string        `json:"name"`
}

func newRequest(t *testing.T, target, body, pathID string) *http.Request {
	t.Helper()

	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), http.MethodPost, target, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), http.MethodPost, target, strings.NewReader(body))
	}

	if pathID != "" {
		r.SetPathValue("id", pathID)
	}

	return r
}

func TestDecode(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		target  string
		body    string
		pathID  string
		sources []httputil.Source
		want    decodeTarget
		wantErr error
	}{
		{
			name:    "path UUID",
			target:  "/x",
			pathID:  testID,
			sources: []httputil.Source{httputil.SourcePath},
			want:    decodeTarget{ID: uuid.MustParse(testID)},
		},
		{
			name:    "malformed path UUID",
			target:  "/x",
			pathID:  "not-a-uuid",
			sources: []httputil.Source{httputil.SourcePath},
			wantErr: httputil.ErrInvalidPath,
		},
		{
			name:    "query time, cursor and int",
			target:  "/x?since=2026-01-02T03:04:05Z&cursor=" + testID + "&limit=5",
			sources: []httputil.Source{httputil.SourceQuery},
			want:    decodeTarget{Since: since, Cursor: uuid.NullUUID{UUID: uuid.MustParse(testID), Valid: true}, Limit: 5},
		},
		{
			name:    "empty cursor stays null",
			target:  "/x?cursor=",
			sources: []httputil.Source{httputil.SourceQuery},
			want:    decodeTarget{},
		},
		{
			name:    "non-RFC3339 time",
			target:  "/x?since=2026-01-02",
			sources: []httputil.Source{httputil.SourceQuery},
			wantErr: httputil.ErrInvalidQuery,
		},
		{
			name:    "malformed cursor",
			target:  "/x?cursor=nope",
			sources: []httputil.Source{httputil.SourceQuery},
			wantErr: httputil.ErrInvalidQuery,
		},
		{
			name:   "body is the default source",
			target: "/x?limit=5",
			body:   `{"name":"alice"}`,
			want:   decodeTarget{Name: "alice"},
		},
		{
			name:    "malformed JSON body",
			target:  "/x",
			body:    `{"name":`,
			wantErr: httputil.ErrInvalidJSON,
		},
		{
			name:    "empty body is not an error",
			target:  "/x",
			sources: []httputil.Source{httputil.SourceBody},
			want:    decodeTarget{},
		},
		{
			name:    "path and body combined",
			target:  "/x",
			body:    `{"name":"alice"}`,
			pathID:  testID,
			sources: []httputil.Source{httputil.SourcePath, httputil.SourceBody},
			want:    decodeTarget{ID: uuid.MustParse(testID), Name: "alice"},
		},
		{
			name:    "unselected sources are ignored",
			target:  "/x?limit=5",
			pathID:  testID,
			sources: []httputil.Source{httputil.SourceBody},
			want:    decodeTarget{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got decodeTarget

			err := httputil.Decode(newRequest(t, tt.target, tt.body, tt.pathID), &got, tt.sources...)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Decode() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && got != tt.want {
				t.Errorf("Decode() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDecodeAndValidate(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `json:"name" validate:"required"`
	}

	tests := []struct {
		name      string
		body      string
		wantValid bool
		wantErr   error
	}{
		{name: "valid", body: `{"name":"alice"}`, wantValid: true},
		{name: "fails validation", body: `{"name":""}`},
		{name: "decode error wins over validation", body: `{`, wantErr: httputil.ErrInvalidJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var req request

			err := httputil.DecodeAndValidate(newRequest(t, "/x", tt.body, ""), &req)

			switch {
			case tt.wantValid:
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
			default:
				if _, ok := errors.AsType[validator.ValidationErrors](err); !ok {
					t.Fatalf("error = %v, want validator.ValidationErrors", err)
				}
			}
		})
	}
}
