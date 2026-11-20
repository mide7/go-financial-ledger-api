package currency

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestCode_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Code
		want bool
	}{
		{name: "uppercase ISO code", in: NGN, want: true},
		{name: "lowercase", in: "ngn", want: false},
		{name: "mixed case", in: "Ngn", want: false},
		{name: "too short", in: "NG", want: false},
		{name: "too long", in: "NGNX", want: false},
		{name: "digits", in: "N1N", want: false},
		{name: "non-ASCII letter", in: "NÉN", want: false},
		{name: "empty", in: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.in.IsValid(); got != tt.want {
				t.Errorf("Code(%q).IsValid() = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatMinor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		minor    int64
		exponent int
		want     string
	}{
		{name: "positive", minor: 12345, exponent: 2, want: "123.45"},
		{name: "negative", minor: -12345, exponent: 2, want: "-123.45"},
		{name: "below one unit", minor: 5, exponent: 2, want: "0.05"},
		{name: "negative below one unit", minor: -5, exponent: 2, want: "-0.05"},
		{name: "zero keeps fraction digits", minor: 0, exponent: 2, want: "0.00"},
		{name: "zero exponent has no point", minor: 100, exponent: 0, want: "100"},
		{name: "three-digit exponent", minor: 1, exponent: 3, want: "0.001"},
		{name: "int64 min", minor: math.MinInt64, exponent: 2, want: "-92233720368547758.08"},
		{name: "int64 max", minor: math.MaxInt64, exponent: 2, want: "92233720368547758.07"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FormatMinor(tt.minor, tt.exponent); got != tt.want {
				t.Errorf("FormatMinor(%d, %d) = %q, want %q", tt.minor, tt.exponent, got, tt.want)
			}
		})
	}
}

// FuzzFormatMinor checks that the output is lossless: removing the decimal point
// gives back the original minor units, with exactly exponent fraction digits.
func FuzzFormatMinor(f *testing.F) {
	f.Add(int64(12345), uint8(2))
	f.Add(int64(-5), uint8(2))
	f.Add(int64(math.MinInt64), uint8(18))

	f.Fuzz(func(t *testing.T, minor int64, exp uint8) {
		exponent := int(exp % 20)
		got := FormatMinor(minor, exponent)

		whole, frac, hasPoint := strings.Cut(got, ".")
		if hasPoint != (exponent > 0) || len(frac) != exponent {
			t.Fatalf("FormatMinor(%d, %d) = %q: want %d fraction digits", minor, exponent, got, exponent)
		}

		back, err := strconv.ParseInt(whole+frac, 10, 64)
		if err != nil || back != minor {
			t.Fatalf("FormatMinor(%d, %d) = %q parses back to %d (err %v)", minor, exponent, got, back, err)
		}
	})
}
