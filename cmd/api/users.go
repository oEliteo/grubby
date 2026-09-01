package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"grubby/internal/auth"
	"grubby/internal/database"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// This file contains handler code for Create, Read, Update, and Delete operations on users.

type UserArgs struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func (u UserArgs) Validate() error {
	if u.Email == "" || u.DisplayName == "" || u.Password == "" {
		return errors.New("email, display_name, and password are all required")
	}
	return nil
}

type UserArgsPartial struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Password    *string `json:"password"`
}

func nullStringFromPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

type UserPrivate struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	IsPremium   bool      `json:"is_premium"`
}

type UserPublic struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
}

func (cfg *apiConfig) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	usrArgs := UserArgs{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&usrArgs); err != nil {
		cfg.log.Warn("error decoding response body", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	hash, err := auth.HashPassword(usrArgs.Password)
	if err != nil {
		cfg.log.Warn("error hashing new user's password", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	dbUsrArgs := database.CreateUserParams{
		ID:             uuid.New(),
		Email:          usrArgs.Email,
		DisplayName:    usrArgs.DisplayName,
		HashedPassword: hash,
		IsPremium:      false,
	}

	dbUsr, err := cfg.db.CreateUser(r.Context(), dbUsrArgs)
	if err != nil {
		cfg.log.Warn("error creating new user in grubby database", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	usrPrivate := UserPrivate{
		ID:          dbUsr.ID,
		CreatedAt:   dbUsr.CreatedAt,
		UpdatedAt:   dbUsr.UpdatedAt,
		DisplayName: dbUsr.DisplayName,
		Email:       dbUsr.Email,
		IsPremium:   dbUsr.IsPremium,
	}

	cfg.respondWithJSON(w, http.StatusCreated, usrPrivate)
}

func (cfg *apiConfig) handleUserGetByIDPublic(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("userID"))
	if err != nil {
		cfg.log.Info("malformed uuid parsed from path", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	dbUsr, err := cfg.db.GetUserByID(r.Context(), userID)
	if err != nil {
		cfg.log.Warn("invalid user id", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	pubUsr := UserPublic{
		ID:          dbUsr.ID,
		DisplayName: dbUsr.DisplayName,
	}

	cfg.respondWithJSON(w, http.StatusOK, pubUsr)
}

func (cfg *apiConfig) handleUserUpdateFull(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := uuid.Parse(r.PathValue("userID"))
	if err != nil {
		cfg.log.Info("malformed uuid parsed from path", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	authUserID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		cfg.log.Info("userID in context is not a uuid")
		cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if authUserID != targetUserID {
		cfg.log.Info("current user is not the owner of the resource")
		cfg.respondWithError(w, http.StatusForbidden, "forbidden")
		return
	}

	params := UserArgs{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		cfg.log.Warn("error decoding request body", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	if err := params.Validate(); err != nil {
		cfg.log.Info("error missing required fields", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		cfg.log.Info("error hashing given password", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	usrUpdateArgs := database.UpdateUserFullParams{
		ID:             targetUserID,
		Email:          params.Email,
		DisplayName:    params.DisplayName,
		HashedPassword: hashedPassword,
	}

	dbUsr, err := cfg.db.UpdateUserFull(r.Context(), usrUpdateArgs)
	if err != nil {
		cfg.log.Info("error updating user record in database", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	usrResponsePrivate := UserPrivate{
		ID:          dbUsr.ID,
		CreatedAt:   dbUsr.CreatedAt,
		UpdatedAt:   dbUsr.UpdatedAt,
		DisplayName: dbUsr.DisplayName,
		Email:       dbUsr.Email,
		IsPremium:   dbUsr.IsPremium,
	}

	cfg.respondWithJSON(w, http.StatusOK, usrResponsePrivate)
}

func (cfg *apiConfig) handleUserUpdatePartial(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := uuid.Parse(r.PathValue("userID"))
	if err != nil {
		cfg.log.Info("malformed uuid parsed from path", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	authUserID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		cfg.log.Info("userID in context is not a uuid")
		cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if authUserID != targetUserID {
		cfg.log.Info("current user is not the owner of the resource")
		cfg.respondWithError(w, http.StatusForbidden, "forbidden")
		return
	}

	params := UserArgsPartial{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil {
		cfg.log.Warn("error decoding request body", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	count := 0
	if params.DisplayName != nil {
		count += 1
	}
	if params.Password != nil {
		count += 1
	}
	if params.Email != nil {
		count += 1
	}

	if count < 1 {
		cfg.log.Info("no payload provided in http patch request")
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	var nullStrPassword sql.NullString
	nullStrDisplayName := nullStringFromPtr(params.DisplayName)
	nullStrEmail := nullStringFromPtr(params.Email)
	if params.Password != nil {
		hashedPassword, err := auth.HashPassword(*params.Password)
		if err != nil {
			cfg.log.Info("error hashing provided password", slog.String("error", err.Error()))
			cfg.respondWithError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		nullStrPassword = sql.NullString{
			String: hashedPassword,
			Valid:  true,
		}
	}

	usrUpdateArgs := database.UpdateUserPartialParams{
		ID:             targetUserID,
		DisplayName:    nullStrDisplayName,
		Email:          nullStrEmail,
		HashedPassword: nullStrPassword,
	}

	dbUsr, err := cfg.db.UpdateUserPartial(r.Context(), usrUpdateArgs)
	if err != nil {
		cfg.log.Info("error patching user record in database", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	usrResponsePrivate := UserPrivate{
		ID:          dbUsr.ID,
		CreatedAt:   dbUsr.CreatedAt,
		UpdatedAt:   dbUsr.UpdatedAt,
		DisplayName: dbUsr.DisplayName,
		Email:       dbUsr.Email,
		IsPremium:   dbUsr.IsPremium,
	}

	cfg.respondWithJSON(w, http.StatusOK, usrResponsePrivate)
}
