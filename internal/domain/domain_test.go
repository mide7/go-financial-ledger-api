package domain

import "testing"

func TestParseEntryType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    EntryType
		wantErr bool
	}{
		{name: "debit", in: "DEBIT", want: EntryTypeDebit},
		{name: "credit", in: "CREDIT", want: EntryTypeCredit},
		{name: "lowercase is rejected", in: "debit", wantErr: true},
		{name: "empty is rejected", in: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseEntryType(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseEntryType(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("ParseEntryType(%q) = %q, want %q", tt.in, got, tt.want)
			}

			if !tt.wantErr && !got.IsValid() {
				t.Errorf("%q.IsValid() = false, want true", got)
			}
		})
	}
}

func TestEntryType_Opposite(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want EntryType }{
		{in: EntryTypeDebit, want: EntryTypeCredit},
		{in: EntryTypeCredit, want: EntryTypeDebit},
		{in: "BOGUS", want: "BOGUS"},
	}

	for _, tt := range tests {
		if got := tt.in.Opposite(); got != tt.want {
			t.Errorf("%q.Opposite() = %q, want %q", tt.in, got, tt.want)
		}
	}
}
