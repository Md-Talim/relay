package app

import (
	"net/http"
)

func (app *Application) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/livez", app.HealthHandler.CheckLiveness)
	mux.HandleFunc("/readyz", app.HealthHandler.CheckReadiness)
	mux.HandleFunc("/health", app.HealthHandler.CheckReadiness)

	mux.HandleFunc("POST /tasks", app.TaskHandler.HandleCreateTask)

	return mux
}
