package validation

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	validate *validator.Validate
)

// Get returns the shared validator, created on first use. Field errors name
// fields the way clients send them: the json tag, else the form tag, else the
// Go field name.
func Get() *validator.Validate {
	once.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		validate.RegisterTagNameFunc(clientFieldName)
	})

	return validate
}

func clientFieldName(f reflect.StructField) string {
	for _, tag := range []string{"json", "form"} {
		name, _, _ := strings.Cut(f.Tag.Get(tag), ",")
		if name != "" && name != "-" {
			return name
		}
	}

	return f.Name
}

// Struct validates s with the shared validator.
func Struct(s any) error {
	return Get().Struct(s)
}
