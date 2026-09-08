package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT           string
	DATABASE_URL   string
	REDIS_HOST     string
	REDIS_PORT     string
	REDIS_PASSWORD string
	REDIS_DB       int
	ADMIN_NAME     string
	ADMIN_EMAIL    string
	ADMIN_PASSWORD string

	FRONTEND_URL string
}

var ENVS = NewConfig()

func NewConfig() *Config {
	godotenv.Load(".env")

	parseRedisDB, err := strconv.ParseInt(getEnv("REDIS_DB", "0"), 10, 64)
	if err != nil {
		slog.Error(fmt.Sprintf("error parsing REDIS_DB: %s", err.Error()))
		parseRedisDB = 0
	}

	return &Config{
		PORT:           getEnv("PORT", ":8080"),
		DATABASE_URL:   getEnv("DATABASE_URL", ""),
		REDIS_HOST:     getEnv("REDIS_HOST", "localhost"),
		REDIS_PORT:     getEnv("REDIS_PORT", "6379"),
		REDIS_PASSWORD: getEnv("REDIS_PASSWORD", ""),
		REDIS_DB:       int(parseRedisDB),
		ADMIN_NAME:     getEnv("ADMIN_NAME", ""),
		ADMIN_EMAIL:    getEnv("ADMIN_EMAIL", ""),
		ADMIN_PASSWORD: getEnv("ADMIN_PASSWORD", ""),
		FRONTEND_URL:   getEnv("FRONTEND_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
