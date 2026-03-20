package db

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := os.Getenv("RELAY_DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("missing RELAY_DATABASE_URL")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
