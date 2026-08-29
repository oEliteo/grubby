package main

import (
	"encoding/json"
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

type UserResponse struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Email       string
	DisplayName string
	IsPremium   bool
}

func (cfg *apiConfig) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	usrArgs := UserArgs{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&decoder); err != nil {
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

	usr := UserResponse{
		ID:          dbUsr.ID,
		CreatedAt:   dbUsr.CreatedAt,
		UpdatedAt:   dbUsr.UpdatedAt,
		DisplayName: dbUsr.DisplayName,
		Email:       dbUsr.Email,
		IsPremium:   dbUsr.IsPremium,
	}

	cfg.respondWithJSON(w, http.StatusCreated, usr)
}
