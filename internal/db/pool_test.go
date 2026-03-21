package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/md-talim/relay/internal/db"
)

func TestOpen_Success(t *testing.T) {
	os.Setenv("RELAY_DATABASE_URL", "postgres://relay:relay@localhost:5433/relay_test?sslmode=disable")
	ctx := context.Background()
	pool, err := db.Open(ctx)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	pool.Close()
}

func TestOpen_MissingEnv(t *testing.T) {
	os.Unsetenv("RELAY_DATABASE_URL")
	ctx := context.Background()
	pool, err := db.Open(ctx)
	if err == nil {
		pool.Close()
		t.Fatalf("expected success, got error: %v", err)
	}
}
