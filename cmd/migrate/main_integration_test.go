//go:build integration

package main

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mide7/go-financial-ledger-api/internal/config"
)

// TestRun_UpDownUp applies every migration, reverts them all, and applies them
// again on a throwaway database. The second "up" fails if any .down.sql file
// leaves something behind (a type, function or trigger), which matters while
// migrations are edited in place.
//
// TEST_DATABASE_URL must allow CREATE DATABASE; the dev compose superuser does.
// Not parallel: it changes the working directory.
func TestRun_UpDownUp(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	tempURL := createTempDatabase(t, adminURL)

	// migrationsSource is relative to the repository root.
	t.Chdir("../..")

	cfg := &config.Config{DatabaseURL: tempURL}
	log := slog.New(slog.DiscardHandler)

	for i, cmd := range []string{cmdUp, cmdDown, cmdUp} {
		if err := run(cmd, cfg, log); err != nil {
			t.Fatalf("step %d (%s): %v", i+1, cmd, err)
		}
	}
}

// createTempDatabase creates an empty database next to the one in adminURL,
// drops it when the test ends, and returns its connection URL.
func createTempDatabase(t *testing.T, adminURL string) string {
	t.Helper()

	ctx := t.Context()

	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	name := "migrate_test_" + uuid.NewString()[:8]
	ident := pgx.Identifier{name}.Sanitize()

	if _, err := conn.Exec(ctx, "CREATE DATABASE "+ident); err != nil {
		_ = conn.Close(ctx)

		t.Fatalf("create database: %v", err)
	}

	t.Cleanup(func() {
		// t.Context() is already cancelled during cleanup.
		cleanupCtx := context.WithoutCancel(ctx)
		if _, err := conn.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+ident+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database %s: %v", name, err)
		}

		_ = conn.Close(cleanupCtx)
	})

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}

	u.Path = "/" + name

	return u.String()
}
