// Package validation validates request structs and formats their errors for clients.
package validation

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

// paramPlaceholder in a tagMessages entry is replaced with the tag's
// parameter, e.g. the 3 in `len=3`.
const paramPlaceholder = "{param}"

var tagMessages = map[string]string{
	"required":  "required",
	"uuid":      "must be a valid UUID",
	"gt":        "must be greater than {param}",
	"gte":       "must be greater than or equal to {param}",
	"lt":        "must be less than {param}",
	"lte":       "must be less than or equal to {param}",
	"len":       "must be exactly {param} characters",
	"oneof":     "must be one of: [{param}]",
	"min":       "must be at least {param}",
	"max":       "must be at most {param}",
	"uppercase": "must be uppercase",
	"lowercase": "must be lowercase",
	"alphanum":  "must be alphanumeric",
	"isdefault": "is not supported yet; omit this field",
}

// FieldError is a client-facing validation failure for one field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FormatValidationErrors converts a go-playground/validator error to a FieldError array.
func FormatValidationErrors(err error) []FieldError {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []FieldError{{Field: "unknown", Message: err.Error()}}
	}

	out := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		out = append(out, FieldError{
			Field:   toLowerCase(fe.Field()),
			Message: msgForTag(fe),
		})
	}

	return out
}

func msgForTag(fe validator.FieldError) string {
	msg, ok := tagMessages[fe.Tag()]
	if !ok {
		return "is invalid"
	}

	return strings.ReplaceAll(msg, paramPlaceholder, fe.Param())
}

func toLowerCase(str string) string {
	return strings.ToLower(str)
}
