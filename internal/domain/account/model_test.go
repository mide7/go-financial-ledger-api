package account

import "testing"

func TestType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Type
		want bool
	}{
		{name: "credit-normal liability", in: TypeUserWallet, want: true},
		{name: "debit-normal asset", in: TypePlatformCash, want: true},
		{name: "lowercase", in: "user_wallet", want: false},
		{name: "empty", in: "", want: false},
		{name: "unknown", in: "SAVINGS", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.in.IsValid(); got != tt.want {
				t.Errorf("Type(%q).IsValid() = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestAllTypes guards the seeder, which creates one system account per entry:
// every entry must be valid and appear once.
func TestAllTypes(t *testing.T) {
	t.Parallel()

	seen := make(map[Type]bool, len(AllTypes))

	for _, typ := range AllTypes {
		if !typ.IsValid() {
			t.Errorf("AllTypes contains invalid type %q", typ)
		}

		if seen[typ] {
			t.Errorf("AllTypes lists %q more than once", typ)
		}

		seen[typ] = true
	}

	if len(AllTypes) != 10 {
		t.Errorf("len(AllTypes) = %d, want 10; update this test and the seeder's account_types when adding a type", len(AllTypes))
	}
}
