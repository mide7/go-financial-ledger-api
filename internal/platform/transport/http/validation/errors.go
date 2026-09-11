package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WriteValidationErrors writes a FieldError array to the response writer.
func WriteValidationErrors(w http.ResponseWriter, err error) {
	details := FormatValidationErrors(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	// Only write the errors part — middleware will wrap it
	_ = json.NewEncoder(w).Encode(details)
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
	switch fe.Tag() {
	case "required":
		return "required"
	case "uuid":
		return "must be a valid UUID"
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: [%s]", fe.Param())
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "uppercase":
		return "must be uppercase"
	case "lowercase":
		return "must be lowercase"
	case "alphanum":
		return "must be alphanumeric"
	default:
		return "is invalid"
	}
}

func toLowerCase(str string) string {
	return strings.ToLower(str)
}
