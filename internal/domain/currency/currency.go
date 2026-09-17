package currency

import "errors"

var ErrInvalidCurrency = errors.New("invalid or unsupported currency code")

type Code string

const (
	NGN Code = "NGN"
	USD Code = "USD"
)

func (c Code) String() string {
	return string(c)
}

func (c Code) IsValid() bool {
	if len(c) != 3 {
		return false
	}
	for i := range 3 {
		if c[i] < 'A' || c[i] > 'Z' {
			return false
		}
	}
	return true
}

type Currency struct {
	Code     Code
	Name     string
	Exponent int // Number of decimal places
	IsActive bool
}
