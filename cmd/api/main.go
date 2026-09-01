package main

import (
	"database/sql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"grubby/internal/database"
	"grubby/internal/logging"
	"grubby/internal/middleware"
	"log/slog"
	"net/http"
	"os"
)

type apiConfig struct {
	db        *database.Queries
	platform  string
	jwtSecret string
	port      string
	amqp      *amqp.Connection
	log       *slog.Logger
}

func main() {
	prettyLogsHandler := logging.NewPrettyLogsHandler(os.Stdout, slog.LevelDebug)
	log := slog.New(prettyLogsHandler)
	err := godotenv.Load()
	if err != nil {
		log.Error("Failed to load environment variables... Exiting...", slog.String("error", err.Error()))
		return
	}

	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	secret := os.Getenv("JWT_SECRET")
	port := os.Getenv("PORT")
	amqpUrl := os.Getenv("AMQP_URL")

	amqpConn, err := amqp.Dial(amqpUrl)
	if err != nil {
		log.Error("Failed to establish connection to RabbitMQ", slog.String("error", err.Error()))
		return
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Error("failed to establish database connection", slog.String("error", err.Error()))
		return
	}

	dbQueries := database.New(db)

	//Initialize application state.

	cfg := &apiConfig{
		db:        dbQueries,
		platform:  platform,
		jwtSecret: secret,
		port:      port,
		amqp:      amqpConn,
		log:       log,
	}

	//Initialize HTTP routes to handler functions.

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", cfg.handleCheckHealth)
	mux.HandleFunc("POST /api/users", cfg.handleUserCreate)
	mux.HandleFunc("GET /api/users/{userID}", cfg.handleUserGetByIDPublic)
	mux.HandleFunc("POST /api/users/{userID}", cfg.Authenticate(cfg.handleUserUpdateFull))
	mux.HandleFunc("PATCH /api/users/{userID}", cfg.Authenticate(cfg.handleUserUpdatePartial))

	//Wrap mux with logger.
	loggedMux := middleware.RequestLogger(log)(mux)

	//Log server start.
	log.Info("HTTP Server Starting...")

	//Listen and serve on port listed in .env
	if err := http.ListenAndServe(cfg.port, loggedMux); err != nil {
		log.Error("server failed", slog.String("error", err.Error()))
	}
}
