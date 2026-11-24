package config

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/mide7/go-financial-ledger-api/internal/logger"
)

// These tests use t.Setenv, so they cannot run in parallel.

func TestFromEnv_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/ledger")

	cfg, err := fromEnv()
	if err != nil {
		t.Fatalf("fromEnv() error = %v", err)
	}

	if cfg.Port != ":8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, ":8080")
	}

	if cfg.RedisDB != 0 {
		t.Errorf("RedisDB = %d, want 0", cfg.RedisDB)
	}
}

func TestFromEnv_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := fromEnv()
	if !errors.Is(err, ErrMissingRequired) {
		t.Fatalf("fromEnv() error = %v, want ErrMissingRequired", err)
	}
}

func TestFromEnv_InvalidRedisDB(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/ledger")
	t.Setenv("REDIS_DB", "not-a-number")

	if _, err := fromEnv(); err == nil {
		t.Fatal("fromEnv() error = nil, want parse error")
	}
}

func TestFromEnv_LoggingDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/ledger")
	t.Setenv("APP_ENV", "") // empty behaves like unset

	cfg, err := fromEnv()
	if err != nil {
		t.Fatalf("fromEnv() error = %v", err)
	}

	// Empty logging values let logger.New apply the per-environment defaults.
	if cfg.AppEnv != logger.EnvDevelopment || cfg.LogLevel != "" || cfg.LogFormat != "" || cfg.LogOutputs != nil {
		t.Errorf("logging config = %q %q %q %v, want development with empty overrides",
			cfg.AppEnv, cfg.LogLevel, cfg.LogFormat, cfg.LogOutputs)
	}
}

func TestFromEnv_InvalidAppEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/ledger")
	t.Setenv("APP_ENV", "staging")

	if _, err := fromEnv(); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("fromEnv() error = %v, want ErrInvalidValue", err)
	}
}

func TestFromEnv_LogOutputs(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "single", raw: "stderr", want: []string{"stderr"}},
		{
			name: "several with spaces",
			raw:  " stdout@json , file:/var/log/app.log ,stderr@text",
			want: []string{"stdout@json", "file:/var/log/app.log", "stderr@text"},
		},
		{name: "blank items dropped", raw: "stderr,,  ,", want: []string{"stderr"}},
		{name: "empty", raw: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://localhost/ledger")
			t.Setenv("LOG_OUTPUTS", tt.raw)

			cfg, err := fromEnv()
			if err != nil {
				t.Fatalf("fromEnv() error = %v", err)
			}

			if !slices.Equal(cfg.LogOutputs, tt.want) {
				t.Errorf("LogOutputs = %q, want %q", cfg.LogOutputs, tt.want)
			}
		})
	}
}

func TestConfig_LoggerOptions(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		AppEnv:     logger.EnvProduction,
		LogLevel:   "warn",
		LogFormat:  logger.FormatJSON,
		LogOutputs: []string{"stdout"},
	}

	got := cfg.LoggerOptions("migrate")
	if got.Env != logger.EnvProduction || got.Level != "warn" || got.Format != logger.FormatJSON ||
		!slices.Equal(got.Outputs, []string{"stdout"}) || got.Service != "migrate" {
		t.Errorf("LoggerOptions() = %+v", got)
	}
}

func TestFromEnv_IdempotencyTTL(t *testing.T) {
	tests := []struct {
		name    string
		raw     *string
		want    time.Duration
		wantErr bool
	}{
		{name: "default", want: 24 * time.Hour},
		{name: "custom", raw: new("90m"), want: 90 * time.Minute},
		{name: "not a duration", raw: new("tomorrow"), wantErr: true},
		{name: "zero", raw: new("0s"), wantErr: true},
		{name: "negative", raw: new("-1h"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://localhost/ledger")

			if tt.raw != nil {
				t.Setenv("IDEMPOTENCY_TTL", *tt.raw)
			}

			cfg, err := fromEnv()
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidValue) {
					t.Fatalf("fromEnv() error = %v, want ErrInvalidValue", err)
				}

				return
			}

			if err != nil || cfg.IdempotencyTTL != tt.want {
				t.Fatalf("fromEnv() = %v, %v; want IdempotencyTTL %v", cfg.IdempotencyTTL, err, tt.want)
			}
		})
	}
}
