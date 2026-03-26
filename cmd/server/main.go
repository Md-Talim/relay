package main

import (
	"net/http"
	"os"
	"time"

	"github.com/md-talim/relay/internal/app"
)

func main() {
	start := time.Now()

	application, err := app.NewApplication(start)
	if err != nil {
		panic(err)
	}
	defer application.DB.Close()

	routes := application.SetupRoutes()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           routes,
		ReadHeaderTimeout: 5 * time.Second,
	}

	application.Logger.Info("http server starting", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		application.Logger.Error("http server failed", "err", err)
		os.Exit(1)
	}
}
