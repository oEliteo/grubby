package main

import (
	"grubby/internal/logging"
	"grubby/internal/middleware"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	prettyLogsHandler := logging.NewPrettyLogsHandler(os.Stdout, slog.LevelDebug)
	log := slog.New(prettyLogsHandler)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	loggedMux := middleware.RequestLogger(log)(mux)

	log.Info("HTTP Server Starting...")

	if err := http.ListenAndServe(":8080", loggedMux); err != nil {
		log.Error("server failed", slog.String("error", err.Error()))
	}
}
