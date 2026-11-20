// Package currency defines currency codes and the registry of supported currencies.
package currency

import (
	"errors"
	"strconv"
	"strings"
)

// ErrInvalidCurrency is returned for a malformed or unsupported currency code.
var ErrInvalidCurrency = errors.New("invalid or unsupported currency code")

// Code is an ISO 4217 alphabetic currency code, such as NGN.
type Code string

// Currency codes seeded by default.
const (
	NGN Code = "NGN"
	USD Code = "USD"
	EUR Code = "EUR"
)

func (c Code) String() string {
	return string(c)
}

// IsValid reports whether c is three uppercase ASCII letters. It does not check support.
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

// Currency describes a currency and how many minor-unit digits it uses.
type Currency struct {
	Code     Code
	Name     string
	Exponent int // Number of decimal places
	IsActive bool
}

// FormatMinor renders an amount in minor units as an exact decimal string with
// exponent fractional digits, e.g. FormatMinor(-12345, 2) == "-123.45".
func FormatMinor(minor int64, exponent int) string {
	digits := strconv.FormatInt(minor, 10)

	sign := ""
	if digits[0] == '-' {
		sign, digits = "-", digits[1:]
	}

	if exponent <= 0 {
		return sign + digits
	}

	if pad := exponent + 1 - len(digits); pad > 0 {
		digits = strings.Repeat("0", pad) + digits
	}

	point := len(digits) - exponent

	return sign + digits[:point] + "." + digits[point:]
}
