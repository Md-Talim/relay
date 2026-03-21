package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/md-talim/relay/internal/api"
	"github.com/md-talim/relay/internal/db"
)

type Application struct {
	DB            *pgxpool.Pool
	Start         time.Time
	Logger        *slog.Logger
	HealthHandler *api.HealthHandler
}

func NewApplication(start time.Time) (*Application, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	pool, err := db.Open(ctx)
	if err != nil {
		logger.Error("failed to create db pool", "err", err)
		os.Exit(1)
	}

	healthHandler := api.NewHealthHandler(start, pool)

	app := &Application{
		DB:            pool,
		Start:         start,
		Logger:        logger,
		HealthHandler: healthHandler,
	}

	return app, nil
}
