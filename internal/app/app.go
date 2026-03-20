package app

import (
	"log/slog"
	"os"
	"time"

	"github.com/md-talim/relay/internal/api"
)

type Application struct {
	Start         time.Time
	Logger        *slog.Logger
	HealthHandler *api.HealthHandler
}

func NewApplication(start time.Time) *Application {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	healthHandler := api.NewHealthHandler()

	return &Application{
		Start:         start,
		Logger:        logger,
		HealthHandler: healthHandler,
	}
}
