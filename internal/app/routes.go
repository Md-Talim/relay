package app

import (
	"net/http"
)

func (app *Application) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/livez", app.HealthHandler.CheckLiveness(app.Start))

	return mux
}
