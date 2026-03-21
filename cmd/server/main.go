package main

import (
	"net/http"
	"os"
	"time"

	"github.com/md-talim/relay/internal/app"
)

func main() {
	start := time.Now()

	app, err := app.NewApplication(start)
	if err != nil {
		panic(err)
	}
	defer app.DB.Close()

	routes := app.SetupRoutes()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           routes,
		ReadHeaderTimeout: 5 * time.Second,
	}

	app.Logger.Info("http server starting", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		app.Logger.Error("http server failed", "err", err)
		os.Exit(1)
	}
}
