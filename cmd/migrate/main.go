package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/md-talim/relay/internal/database/migrations"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	dbURL := os.Getenv("RELAY_DATABASE_URL")
	if dbURL == "" {
		logger.Error("missing RELAY_DATABASE_URL")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Error("failed to create db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	mctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := migrations.RunMigrations(mctx, pool, "internal/database/migrations"); err != nil {
		logger.Error("migration run failed", "err", err)
		os.Exit(1)
	}

	logger.Info("migration run complete")
}
