package logger_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/logger"
)

func TestDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		env         string
		wantLevel   string
		wantFormat  string
		wantOutputs []string
	}{
		{name: "development", env: logger.EnvDevelopment, wantLevel: "debug", wantFormat: logger.FormatText, wantOutputs: []string{"stderr"}},
		{name: "production", env: logger.EnvProduction, wantLevel: "info", wantFormat: logger.FormatJSON, wantOutputs: []string{"stdout"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			level, format, outputs := logger.Defaults(tt.env)
			if level != tt.wantLevel || format != tt.wantFormat || !slices.Equal(outputs, tt.wantOutputs) {
				t.Errorf("Defaults(%q) = %q, %q, %v; want %q, %q, %v",
					tt.env, level, format, outputs, tt.wantLevel, tt.wantFormat, tt.wantOutputs)
			}
		})
	}
}

func TestNew_InvalidOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts logger.Options
	}{
		{name: "unknown env", opts: logger.Options{Env: "staging"}},
		{name: "empty env", opts: logger.Options{}},
		{name: "unknown level", opts: logger.Options{Env: logger.EnvDevelopment, Level: "verbose"}},
		{name: "unknown format", opts: logger.Options{Env: logger.EnvDevelopment, Format: "xml"}},
		{name: "unknown output", opts: logger.Options{Env: logger.EnvDevelopment, Outputs: []string{"syslog"}}},
		{name: "file without path", opts: logger.Options{Env: logger.EnvDevelopment, Outputs: []string{"file:"}}},
		{name: "unknown output format", opts: logger.Options{Env: logger.EnvDevelopment, Outputs: []string{"stdout@yaml"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := logger.New(tt.opts); !errors.Is(err, logger.ErrInvalidOption) {
				t.Errorf("New(%+v) error = %v, want ErrInvalidOption", tt.opts, err)
			}
		})
	}
}

func TestNew_StdStreams(t *testing.T) {
	t.Parallel()

	for _, out := range []string{"stdout", "stderr", "stdout@json", "stderr@text", " stdout "} {
		t.Run(out, func(t *testing.T) {
			t.Parallel()

			log, closeLog, err := logger.New(logger.Options{Env: logger.EnvProduction, Outputs: []string{out}})
			if err != nil || log == nil {
				t.Fatalf("New(output %q) = %v, %v", out, log, err)
			}

			if err := closeLog(); err != nil {
				t.Errorf("close: %v", err)
			}
		})
	}
}

func TestNew_FileOutputs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "app.json.log")
	textPath := filepath.Join(dir, "app.text.log")

	log, closeLog, err := logger.New(logger.Options{
		Env:     logger.EnvProduction,
		Level:   "warn",
		Service: "ledger-api",
		Outputs: []string{"file:" + jsonPath, "file:" + textPath + "@text"},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	log.Info("below level, dropped")
	log.Warn("balance discrepancy detected", "account_id", "a1")

	if err := closeLog(); err != nil {
		t.Fatalf("close: %v", err)
	}

	jsonOut := readFile(t, jsonPath)
	if strings.Count(jsonOut, "\n") != 1 {
		t.Fatalf("JSON output has %d lines, want 1 (info must be filtered):\n%s", strings.Count(jsonOut, "\n"), jsonOut)
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &record); err != nil {
		t.Fatalf("JSON output is not JSON: %v\n%s", err, jsonOut)
	}

	if record["msg"] != "balance discrepancy detected" || record["service"] != "ledger-api" ||
		record["account_id"] != "a1" || record["level"] != "WARN" {
		t.Errorf("JSON record = %v", record)
	}

	if _, hasSource := record["source"]; hasSource {
		t.Error("production records must not include source locations")
	}

	textOut := readFile(t, textPath)
	if !strings.Contains(textOut, `msg="balance discrepancy detected"`) || !strings.Contains(textOut, "service=ledger-api") {
		t.Errorf("text output = %q", textOut)
	}
}

func TestNew_DevelopmentAddsSource(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "dev.log")

	log, closeLog, err := logger.New(logger.Options{Env: logger.EnvDevelopment, Outputs: []string{"file:" + path + "@json"}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	log.Debug("debug is on in development")

	if err := closeLog(); err != nil {
		t.Fatalf("close: %v", err)
	}

	out := readFile(t, path)
	if !strings.Contains(out, `"source"`) || !strings.Contains(out, "debug is on in development") {
		t.Errorf("development output = %s, want a debug record with source", out)
	}
}

func TestNew_ClosesOpenedFilesOnError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "first.log")

	// The second output is invalid, so New must fail and release the first file.
	_, _, err := logger.New(logger.Options{Env: logger.EnvProduction, Outputs: []string{"file:" + path, "nowhere"}})
	if !errors.Is(err, logger.ErrInvalidOption) {
		t.Fatalf("New() error = %v, want ErrInvalidOption", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("first output was not opened before the failure: %v", statErr)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path) //nolint:gosec // path is a test temp file
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(b)
}
