package domain

import "fmt"

type EntryType string

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

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

func (t EntryType) IsValid() bool {
	switch t {
	case EntryTypeDebit, EntryTypeCredit:
		return true
	default:
		return false
	}
}
