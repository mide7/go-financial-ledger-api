// Package config loads application settings from environment variables.
package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"github.com/mide7/go-financial-ledger-api/internal/logger"
)

// ErrMissingRequired is returned when a required environment variable is unset or empty.
var ErrMissingRequired = errors.New("missing required environment variable")

// ErrInvalidValue is returned when an environment variable has an unsupported value.
var ErrInvalidValue = errors.New("invalid environment variable")

// Config holds settings read from the environment.
type Config struct {
	// AppEnv is logger.EnvDevelopment (default) or logger.EnvProduction.
	AppEnv string
	// LogLevel, LogFormat and LogOutputs are passed to logger.New as given;
	// empty values take the defaults for AppEnv (see logger.Defaults).
	LogLevel   string
	LogFormat  string
	LogOutputs []string

	Port          string
	DatabaseURL   string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	AdminName     string
	AdminEmail    string
	AdminPassword string
	FrontendURL   string
}

// Load reads a local .env file if one exists, then builds a Config from the
// environment. Variables already set in the process take precedence over .env.
func Load() (*Config, error) {
	// A missing .env is normal outside local development, so the error is ignored.
	_ = godotenv.Load(".env")

	return fromEnv()
}

func fromEnv() (*Config, error) {
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_DB: %w", err)
	}

	cfg := &Config{
		AppEnv:        cmp.Or(getEnv("APP_ENV", ""), logger.EnvDevelopment), // unset or empty → development
		LogLevel:      getEnv("LOG_LEVEL", ""),
		LogFormat:     getEnv("LOG_FORMAT", ""),
		LogOutputs:    splitList(getEnv("LOG_OUTPUTS", "")),
		Port:          getEnv("PORT", ":8080"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,
		AdminName:     getEnv("ADMIN_NAME", ""),
		AdminEmail:    getEnv("ADMIN_EMAIL", ""),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		FrontendURL:   getEnv("FRONTEND_URL", ""),
	}

	if cfg.AppEnv != logger.EnvDevelopment && cfg.AppEnv != logger.EnvProduction {
		return nil, fmt.Errorf("%w: APP_ENV=%q (want %s or %s)", ErrInvalidValue, cfg.AppEnv, logger.EnvDevelopment, logger.EnvProduction)
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("%w: DATABASE_URL", ErrMissingRequired)
	}

	return cfg, nil
}

// LoggerOptions returns the logger configuration for a command named service.
func (c *Config) LoggerOptions(service string) logger.Options {
	return logger.Options{
		Env:     c.AppEnv,
		Level:   c.LogLevel,
		Format:  c.LogFormat,
		Outputs: c.LogOutputs,
		Service: service,
	}
}

// splitList splits a comma-separated value, dropping blank items.
func splitList(v string) []string {
	var out []string

	for item := range strings.SplitSeq(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}

	return out
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
