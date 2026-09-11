package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/form/v4"
	"github.com/mide7/go-financial-ledger-api/internal/validator"
)

var (
	ErrInvalidJSON  = errors.New("invalid JSON body")
	ErrInvalidQuery = errors.New("invalid query parameters")
	ErrInvalidPath  = errors.New("invalid path parameters")
	formDecoder     = form.NewDecoder()
)

type Source uint8

const (
	SourceBody Source = 1 << iota
	SourceQuery
	SourcePath
)

// Decode parses the request body into v.
func Decode(r *http.Request, v any, sources ...Source) error {
	// if no sources are specified, default to body
	if len(sources) == 0 {
		sources = []Source{SourceBody}
	}

	var combinedSources Source
	for _, s := range sources {
		combinedSources |= s
	}

	// 1. Decode Path parameters
	if combinedSources&SourcePath != 0 {
		pathMap := make(map[string][]string)
		// Extract path values into map for formDecoder
		if r.URL != nil {
			//NOTE: add target path keys as needed
			for _, key := range []string{"id"} {
				if val := r.PathValue(key); val != "" {
					pathMap[key] = []string{val}
				}
			}
			if len(pathMap) > 0 {
				if err := formDecoder.Decode(v, pathMap); err != nil {
					return fmt.Errorf("%w: %v", ErrInvalidPath, err)
				}
			}
		}
	}

	// 2. Decode Query parameters
	if combinedSources&SourceQuery != 0 {
		if err := formDecoder.Decode(v, r.URL.Query()); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
		}
	}

	// 3. Decode JSON Body
	if combinedSources&SourceBody != 0 && r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(v); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
		}
	}

	return nil
}

// DecodeAndValidate decodes the JSON body and immediately runs validation on it.
func DecodeAndValidate(r *http.Request, v any, sources ...Source) error {
	if err := Decode(r, v, sources...); err != nil {
		return err
	}
	if err := validator.Struct(v); err != nil {
		return err
	}
	return nil
}
