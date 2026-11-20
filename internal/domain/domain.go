// Package domain holds types shared by the ledger's domain packages.
package domain

import (
	"errors"
	"fmt"
)

// ErrNotImplemented is returned by operations that exist in the API but have
// no persistence behind them yet. The HTTP layer maps it to 501.
var ErrNotImplemented = errors.New("not implemented")

// EntryType is the side of the ledger an entry posts to.
type EntryType string

// Entry types.
const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

// ParseEntryType converts s to an EntryType. Matching is case-sensitive.
func ParseEntryType(s string) (EntryType, error) {
	switch s {
	case "DEBIT":
		return EntryTypeDebit, nil
	case "CREDIT":
		return EntryTypeCredit, nil
	default:
		return "", fmt.Errorf("invalid entry type: %q", s)
	}
}

// IsValid reports whether t is a known entry type.
func (t EntryType) IsValid() bool {
	switch t {
	case EntryTypeDebit, EntryTypeCredit:
		return true
	default:
		return false
	}
}
