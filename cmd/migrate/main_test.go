package main

import (
	"errors"
	"testing"
)

func TestParseCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr error
	}{
		{name: "up", args: []string{cmdUp}, want: cmdUp},
		{name: "down", args: []string{cmdDown}, want: cmdDown},
		{name: "no arguments", args: nil, wantErr: errUsage},
		{name: "unknown command", args: []string{"sideways"}, wantErr: errUsage},
		{name: "wrong case", args: []string{"UP"}, wantErr: errUsage},
		{name: "extra argument", args: []string{cmdDown, "all"}, wantErr: errUsage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCommand(tt.args)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseCommand(%q) error = %v, want %v", tt.args, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("parseCommand(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// TestStart_RejectsBadCommandBeforeConfig leaves DATABASE_URL empty, which
// would fail config loading with exit code 1: getting the usage code 2 proves
// the command is validated first. Not parallel because it sets an env var.
func TestStart_RejectsBadCommandBeforeConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{name: "unknown command", args: []string{"sideways"}, wantCode: 2},
		{name: "no command", args: nil, wantCode: 2},
		{name: "valid command, missing config", args: []string{cmdUp}, wantCode: 1},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share the parent's t.Setenv
		t.Run(tt.name, func(t *testing.T) {
			if got := start(tt.args); got != tt.wantCode {
				t.Errorf("start(%q) = %d, want %d", tt.args, got, tt.wantCode)
			}
		})
	}
}
