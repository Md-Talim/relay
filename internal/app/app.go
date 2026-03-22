package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/md-talim/relay/internal/api"
	"github.com/md-talim/relay/internal/db"
	"github.com/md-talim/relay/internal/store"
)

type Application struct {
	DB            *pgxpool.Pool
	Start         time.Time
	Logger        *slog.Logger
	HealthHandler *api.HealthHandler
	TaskHandler   *api.TaskHandler
}

func NewApplication(start time.Time) (*Application, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	pool, err := db.Open(ctx)
	if err != nil {
		logger.Error("failed to create db pool", "err", err)
		os.Exit(1)
	}

	taskStore := store.NewTaskStore(pool)

	healthHandler := api.NewHealthHandler(start, pool)
	taskHandler := api.NewTaskHandler(taskStore, logger)

	app := &Application{
		DB:            pool,
		Start:         start,
		Logger:        logger,
		HealthHandler: healthHandler,
		TaskHandler:   taskHandler,
	}

	return app, nil
}
