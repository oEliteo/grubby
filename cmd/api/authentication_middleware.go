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
			cfg.log.Warn("error retrieving bearer token... unauthorized : 401", slog.String("error", err.Error()))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			cfg.log.Warn("received invalid token... unauthorized : 401", slog.String("error", err.Error()))
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		handler(w, r.WithContext(ctx))
	})
}
