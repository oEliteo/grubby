package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"grubby/internal/auth"
	"grubby/internal/database"
	"grubby/internal/logging"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func newTestConfig() (*apiConfig, *sql.DB, error) {
	prettyLogsHandler := logging.NewPrettyLogsHandler(os.Stdout, slog.LevelDebug)
	log := slog.New(prettyLogsHandler)
	err := godotenv.Load()
	if err != nil {
		err = godotenv.Load("../../.env")
		if err != nil {
			return &apiConfig{}, &sql.DB{}, err
		}
	}

	dbURL := os.Getenv("DBTEST_URL")
	platform := os.Getenv("PLATFORM_TEST")
	secret := os.Getenv("TEST_SECRET")
	port := os.Getenv("PORT")
	amqpUrl := os.Getenv("AMQPTEST_URL")

	amqpConn, err := amqp.Dial(amqpUrl)
	if err != nil {
		log.Error("Failed to establish connection to RabbitMQ", slog.String("error", err.Error()))
		return &apiConfig{}, &sql.DB{}, err
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Error("failed to establish database connection", slog.String("error", err.Error()))
		return &apiConfig{}, &sql.DB{}, err
	}

	if err := db.Ping(); err != nil {
		return &apiConfig{}, &sql.DB{}, err
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

	return cfg, db, nil
}

func TestHandleUserCreate(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	resetTestDB(t, db)

	usrArgs := UserArgs{
		Email:       "johnsmith@test.com",
		DisplayName: "john_smith07",
		Password:    "easy123!",
	}

	data, err := json.Marshal(usrArgs)
	if err != nil {
		t.Fatalf("error marshalling json payload: %v\n", err)
	}

	reader := bytes.NewReader(data)

	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)

	rr := httptest.NewRecorder()

	cfg.handleUserCreate(rr, request)

	if rr.Code != http.StatusCreated {
		t.Fatalf("incorrect status code found in response: %v\nbody: %v\n", rr.Code, rr.Body.String())
	}

	usrPrivate := UserPrivate{}
	if err := json.Unmarshal(rr.Body.Bytes(), &usrPrivate); err != nil {
		t.Fatalf("unable to unmarshal response body: %v\n", err)
	}

	if usrPrivate.DisplayName != "john_smith07" {
		t.Fatalf("expected: %v got: %v\n", usrArgs.DisplayName, usrPrivate.DisplayName)
	}

	if usrPrivate.Email != usrArgs.Email {
		t.Fatalf("expected: %v got: %v\n", usrArgs.Email, usrPrivate.Email)
	}

	if usrPrivate.ID == uuid.Nil {
		t.Fatalf("expected: new valid uuid got: %v\n", usrPrivate.ID)
	}

	dbUsr, err := cfg.db.GetUserByID(request.Context(), usrPrivate.ID)
	if err != nil {
		t.Fatalf("failed to retrieve user from db: %v\n", err)
	}

	ok, err := auth.CheckPasswordHash(usrArgs.Password, dbUsr.HashedPassword)
	if !ok || err != nil {
		t.Fatalf("password hash invalid or error expected: true got: %v\nerror: %v\n", ok, err)
	}
}

func resetTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec("TRUNCATE TABLE users CASCADE;")
	if err != nil {
		t.Fatalf("failed to reset test database: %v\n", err)
	}
}
