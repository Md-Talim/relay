package tests

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/md-talim/relay/internal/db"
	"github.com/md-talim/relay/internal/db/migrations"
)

func TestRunMigrations(t *testing.T) {
	os.Setenv("RELAY_DATABASE_URL", "postgres://relay:relay@localhost:5433/relay_test?sslmode=disable")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	defer pool.Close()

	logger := slog.Default()
	err = migrations.RunMigrations(ctx, pool, logger, "../internal/db/migrations")
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
}
