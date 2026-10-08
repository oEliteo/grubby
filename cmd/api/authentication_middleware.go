package main

import (
	"context"
	"grubby/internal/auth"
	"log/slog"
	"net/http"
)

type ctxKey string

const userIDKey ctxKey = "userID"

func (cfg *apiConfig) Authenticate(handler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			cfg.log.Info("error retrieving bearer token", slog.String("error", err.Error()))
			cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			cfg.log.Info("error validating bearer token", slog.String("error", err.Error()))
			cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		dbUsr, err := cfg.db.GetUserByID(r.Context(), userID)
		if err != nil {
			cfg.log.Info("error getting user from database, cannot verify account is active", slog.String("error", err.Error()))
			cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if dbUsr.DeletedAt.Valid {
			cfg.log.Info("this account has been deleted and cannot be authenticated", slog.String("error", "account deactivated"))
			cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		handler(w, r.WithContext(ctx))
	})
}
