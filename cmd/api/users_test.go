package main

import (
	"bytes"
	"context"
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
	"time"

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

	defer resetTestDB(t, db)

	createNewUser(t, db, cfg)
}

func TestHandleGetUserByIDPublic(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	data, err := json.Marshal(newUsr)
	if err != nil {
		t.Fatalf("error marshalling json payload: %v\n", err)
	}

	reader := bytes.NewReader(data)

	request := httptest.NewRequest(http.MethodGet, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())

	rr := httptest.NewRecorder()

	cfg.handleUserGetByIDPublic(rr, request)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, rr.Code)
	}

	returnedUsr := UserPublic{}
	if err := json.Unmarshal(rr.Body.Bytes(), &returnedUsr); err != nil {
		t.Fatalf("unable to unmasrhal returned user: %v\n", err)
	}

	if returnedUsr.ID != newUsr.ID {
		t.Fatalf("expected %v got %v", newUsr.ID, returnedUsr.ID)
	}

	if returnedUsr.DisplayName != newUsr.DisplayName {
		t.Fatalf("expected %v got %v", newUsr.DisplayName, returnedUsr.DisplayName)
	}
}

func TestHandleUserUpdateFullBadRequest(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	type badStruct struct {
		Payload string    `json:"payload"`
		Array   *[]string `json:"array"`
	}

	tmp := make([]string, 3)
	tmp[0] = "something"
	tmp[1] = "bad"
	tmp[2] = "2"

	updateUsr := badStruct{
		Payload: "something bad",
		Array:   &tmp,
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", dbUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)
	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullMalformedUUIDInContext(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "noobmail@test.com",
		DisplayName: "noobname07",
		Password:    "noobpass123!",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	ctx := context.WithValue(context.Background(), userIDKey, "not a uuid")
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", dbUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+"this is not a token lol")

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusUnauthorized != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleUserUpdateFullForbidden(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "noobmail@test.com",
		DisplayName: "noobname07",
		Password:    "noobpass123!",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", dbUsr.ID.String())
	token, err := auth.MakeJWT(uuid.New(), cfg.jwtSecret, time.Hour)
	if err != nil {
		t.Fatalf("failed to make new token")
	}
	request.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusForbidden != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusForbidden, rr.Code)
	}
}

func TestHandleUserUpdateFullMalformedUUIDInPath(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "noobmail@test.com",
		DisplayName: "",
		Password:    "noobpass123!",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", "Not a UUID")
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)
	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullAllButDisplayName(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "noobmail@test.com",
		DisplayName: "",
		Password:    "noobpass123!",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)
	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullOnlyPassword(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "",
		DisplayName: "",
		Password:    "noobpass123",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullOnlyDisplayName(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "",
		DisplayName: "noobguy777",
		Password:    "",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullOnlyEmail(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "noobmail@test.com",
		DisplayName: "",
		Password:    "",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)
	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullAllButEmail(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "",
		DisplayName: "noobname07",
		Password:    "noobpass123!",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullAllButPassword(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "noobmail@test.com",
		DisplayName: "noobname07",
		Password:    "",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFullNoCredentials(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "",
		DisplayName: "",
		Password:    "",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)
	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)
	handler(rr, request)

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserUpdateFull(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	updateUsr := UserArgs{
		Email:       "newtest@gmail.com",
		DisplayName: "newtest_guy07",
		Password:    "newtestpass123!",
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdateFull)

	handler(rr, request)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %v got %v", http.StatusOK, rr.Code)
	}

	updatedUsr := UserPrivate{}
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedUsr); err != nil {
		t.Fatalf("error unmarshalling response body: %v\n", err)
	}

	if updatedUsr.ID != dbUsr.ID {
		t.Fatalf("expected %v got %v", updatedUsr.ID, dbUsr.ID)
	}

	if updatedUsr.DisplayName == dbUsr.DisplayName {
		t.Fatalf("did not expect %v got %v", updatedUsr.DisplayName, dbUsr.DisplayName)
	}

	dbUsrNew, err := cfg.db.GetUserByID(context.Background(), dbUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving updated user from database: %v\n", err)
	}

	if dbUsrNew.HashedPassword == dbUsr.HashedPassword {
		t.Fatalf("did not expect %v got %v", dbUsrNew.HashedPassword, dbUsr.HashedPassword)
	}
}

func TestHandleUserUpdatePartialPasswordOnly(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	pass := "newpass123!!!"
	var email, displayName, password *string
	email = nil
	displayName = nil
	password = &pass

	updateUsr := UserArgsPartial{
		Email:       email,
		DisplayName: displayName,
		Password:    password,
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPatch, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdatePartial)

	handler(rr, request)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %v got %v", http.StatusOK, rr.Code)
	}

	updatedUsr := UserPrivate{}
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedUsr); err != nil {
		t.Fatalf("error unmarshalling response body: %v\n", err)
	}

	if updatedUsr.ID != newUsr.ID {
		t.Fatalf("expected %v got %v", newUsr.ID, updatedUsr.ID)
	}

	if updatedUsr.Email != newUsr.Email {
		t.Fatalf("expected %v got %v", newUsr.Email, updatedUsr.Email)
	}

	if updatedUsr.DisplayName != newUsr.DisplayName {
		t.Fatalf("expected %v got %v", newUsr.DisplayName, updatedUsr.DisplayName)
	}

	dbUsrNew, err := cfg.db.GetUserByID(context.Background(), dbUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving updated user from database: %v\n", err)
	}

	if dbUsrNew.HashedPassword == dbUsr.HashedPassword {
		t.Fatalf("did not expect %v got %v", dbUsr.HashedPassword, dbUsrNew.HashedPassword)
	}
}

func TestHandleUserUpdatePartialMultiValue(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	mail := "newmail@newmail.com"
	name := "newname123"
	var email, displayName, password *string
	email = &mail
	displayName = &name
	password = nil

	updateUsr := UserArgsPartial{
		Email:       email,
		DisplayName: displayName,
		Password:    password,
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPatch, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdatePartial)

	handler(rr, request)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %v got %v", http.StatusOK, rr.Code)
	}

	updatedUsr := UserPrivate{}
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedUsr); err != nil {
		t.Fatalf("error unmarshalling response body: %v\n", err)
	}

	if updatedUsr.ID != newUsr.ID {
		t.Fatalf("expected %v got %v", newUsr.ID, updatedUsr.ID)
	}

	if updatedUsr.Email != *updateUsr.Email {
		t.Fatalf("expected %v got %v", *updateUsr.Email, updatedUsr.Email)
	}

	if updatedUsr.DisplayName != *updateUsr.DisplayName {
		t.Fatalf("expected %v got %v", *updateUsr.DisplayName, updatedUsr.DisplayName)
	}

	dbUsrNew, err := cfg.db.GetUserByID(context.Background(), dbUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving updated user from database: %v\n", err)
	}

	if dbUsrNew.HashedPassword != dbUsr.HashedPassword {
		t.Fatalf("expected %v got %v", dbUsr.HashedPassword, dbUsrNew.HashedPassword)
	}
}

func TestHandleUserUpdatePartialEmailOnly(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	mail := "newmail@newmail.com"
	var email, displayName, password *string
	email = &mail
	displayName = nil
	password = nil

	updateUsr := UserArgsPartial{
		Email:       email,
		DisplayName: displayName,
		Password:    password,
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPatch, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdatePartial)

	handler(rr, request)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %v got %v", http.StatusOK, rr.Code)
	}

	updatedUsr := UserPrivate{}
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedUsr); err != nil {
		t.Fatalf("error unmarshalling response body: %v\n", err)
	}

	if updatedUsr.ID != newUsr.ID {
		t.Fatalf("expected %v got %v", newUsr.ID, updatedUsr.ID)
	}

	if updatedUsr.Email != *updateUsr.Email {
		t.Fatalf("expected %v got %v", *updateUsr.Email, updatedUsr.Email)
	}

	if updatedUsr.DisplayName != newUsr.DisplayName {
		t.Fatalf("expected %v got %v", newUsr.DisplayName, updatedUsr.DisplayName)
	}

	dbUsrNew, err := cfg.db.GetUserByID(context.Background(), dbUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving updated user from database: %v\n", err)
	}

	if dbUsrNew.HashedPassword != dbUsr.HashedPassword {
		t.Fatalf("expected %v got %v", dbUsr.HashedPassword, dbUsrNew.HashedPassword)
	}
}

func TestHandleUserUpdatePartialUsernameOnly(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	name := "newname123"
	var email, displayName, password *string
	email = nil
	displayName = &name
	password = nil

	updateUsr := UserArgsPartial{
		Email:       email,
		DisplayName: displayName,
		Password:    password,
	}

	data, err := json.Marshal(updateUsr)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	dbUsr, err := cfg.db.GetUserByID(context.Background(), newUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving user from database: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPatch, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserUpdatePartial)

	handler(rr, request)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %v got %v", http.StatusOK, rr.Code)
	}

	updatedUsr := UserPrivate{}
	if err := json.Unmarshal(rr.Body.Bytes(), &updatedUsr); err != nil {
		t.Fatalf("error unmarshalling response body: %v\n", err)
	}

	if updatedUsr.ID != newUsr.ID {
		t.Fatalf("expected %v got %v", newUsr.ID, updatedUsr.ID)
	}

	if updatedUsr.Email != newUsr.Email {
		t.Fatalf("expected %v got %v", *updateUsr.Email, updatedUsr.Email)
	}

	if updatedUsr.DisplayName != *updateUsr.DisplayName {
		t.Fatalf("expected %v got %v", *updateUsr.DisplayName, updatedUsr.DisplayName)
	}

	dbUsrNew, err := cfg.db.GetUserByID(context.Background(), dbUsr.ID)
	if err != nil {
		t.Fatalf("error retrieving updated user from database: %v\n", err)
	}

	if dbUsrNew.HashedPassword != dbUsr.HashedPassword {
		t.Fatalf("expected %v got %v", dbUsr.HashedPassword, dbUsrNew.HashedPassword)
	}
}

func resetTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec("TRUNCATE TABLE users CASCADE;")
	if err != nil {
		t.Fatalf("failed to reset test database: %v\n", err)
	}
}

func TestHandleUserCreateMissingField1(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	_ = db
	usrArgs := UserArgs{
		Email:       "",
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

	if http.StatusBadRequest != rr.Code {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserDeleteMalformedUUIDInToken(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	data := make([]byte, 0)
	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodDelete, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+"this is not a token lol")

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserDelete)

	handler(rr, request)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %v got %v", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleUserDeleteMalformedUUIDInPath(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	data := make([]byte, 0)
	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodDelete, "/api/users", reader)
	request.SetPathValue("userID", "this is not a uuid lol")
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserDelete)

	handler(rr, request)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleUserDeleteForbidden(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	data := make([]byte, 0)
	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodDelete, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	jwt, err := auth.MakeJWT(uuid.New(), cfg.jwtSecret, time.Hour)
	if err != nil {
		t.Fatalf("error creating new token")
	}

	request.Header.Set("Authorization", "Bearer "+jwt)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserDelete)

	handler(rr, request)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %v got %v", http.StatusForbidden, rr.Code)
	}
}

func TestHandleUserDelete(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	newUsr := createNewUser(t, db, cfg)

	defer resetTestDB(t, db)

	data := make([]byte, 0)
	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodDelete, "/api/users", reader)
	request.SetPathValue("userID", newUsr.ID.String())
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()

	handler := cfg.Authenticate(cfg.handleUserDelete)

	handler(rr, request)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %v got %v", http.StatusNoContent, rr.Code)
	}
}

func createNewUser(t *testing.T, db *sql.DB, cfg *apiConfig) UserPrivate {
	t.Helper()

	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}

	_ = db
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

	if usrPrivate.Token == "" {
		t.Fatalf("unable to create jwt")
	}

	if usrPrivate.RefreshToken == "" {
		t.Fatalf("unable to create refresh token")
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

	return usrPrivate
}
