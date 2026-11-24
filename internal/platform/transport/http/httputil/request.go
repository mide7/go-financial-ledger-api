// Package httputil decodes requests and writes responses for the HTTP handlers.
package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/validation"
)

// Decoding errors, wrapped around the underlying decoder error.
var (
	ErrInvalidJSON  = errors.New("invalid JSON body")
	ErrInvalidQuery = errors.New("invalid query parameters")
	ErrInvalidPath  = errors.New("invalid path parameters")
	formDecoder     = newFormDecoder()
)

// pathKeys lists the route wildcards decodePath reads; add keys as routes need them.
var pathKeys = []string{"id"}

// Source is a bit flag naming a part of the request to decode from.
type Source uint8

// Request sources. Combine them by passing several to Decode.
const (
	SourceBody Source = 1 << iota
	SourceQuery
	SourcePath
)

// Decode fills v from the given request sources, or from the JSON body when none are given.
func Decode(r *http.Request, v any, sources ...Source) error {
	if len(sources) == 0 {
		sources = []Source{SourceBody}
	}

	var combinedSources Source
	for _, s := range sources {
		combinedSources |= s
	}

	if combinedSources&SourcePath != 0 {
		if err := decodePath(r, v); err != nil {
			return err
		}
	}

	if combinedSources&SourceQuery != 0 {
		if err := formDecoder.Decode(v, r.URL.Query()); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidQuery, err)
		}
	}

	hasBody := r.Body != nil && r.ContentLength > 0
	if combinedSources&SourceBody != 0 && hasBody {
		if err := json.NewDecoder(r.Body).Decode(v); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidJSON, err)
		}
	}

	return nil
}

// DecodeAndValidate runs Decode with the same sources, then validates v's struct tags.
func DecodeAndValidate(r *http.Request, v any, sources ...Source) error {
	if err := Decode(r, v, sources...); err != nil {
		return err
	}

	if err := validation.Struct(v); err != nil {
		return err
	}

	return nil
}

func decodePath(r *http.Request, v any) error {
	if r.URL == nil {
		return nil
	}

	pathMap := make(map[string][]string, len(pathKeys))

	for _, key := range pathKeys {
		if val := r.PathValue(key); val != "" {
			pathMap[key] = []string{val}
		}
	}

	if len(pathMap) == 0 {
		return nil
	}

	if err := formDecoder.Decode(v, pathMap); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidPath, err)
	}

	return nil
}

// newFormDecoder returns a path/query decoder that understands the non-primitive
// types used in DTOs. Without these, uuid.UUID is decoded as a [16]byte array
// and every UUID string is rejected.
func newFormDecoder() *form.Decoder {
	d := form.NewDecoder()

	d.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return uuid.Parse(vals[0])
	}, uuid.UUID{})

	d.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		if vals[0] == "" {
			return uuid.NullUUID{}, nil
		}

		id, err := uuid.Parse(vals[0])

		return uuid.NullUUID{UUID: id, Valid: err == nil}, err
	}, uuid.NullUUID{})

	d.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		return time.Parse(time.RFC3339, vals[0])
	}, time.Time{})

	return d
}
