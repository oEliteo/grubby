package main

import (
	"database/sql"
	"encoding/json"
	"grubby/internal/database"
	"grubby/internal/models"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// This file contains handler code for Create, Read, Update, and Delete operations on recipes.
func newNullStringFromPtr(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{
			String: *s,
			Valid:  true,
		}
	} else {
		return sql.NullString{}
	}
}

func (cfg *apiConfig) handleRecipeCreate(w http.ResponseWriter, r *http.Request) {
	authUserID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		cfg.log.Warn("userID in context is not a uuid")
		cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	recipeArgs := models.RecipeArgs{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&recipeArgs); err != nil {
		cfg.log.Warn("error decoding request body", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	ownerID := uuid.NullUUID{
		Valid: true,
		UUID:  authUserID,
	}

	recipeID := uuid.New()

	recipeParams := database.CreateRecipeParams{
		ID:           recipeID,
		UserID:       ownerID,
		Title:        recipeArgs.Title,
		Description:  newNullStringFromPtr(recipeArgs.Description),
		IsPublic:     recipeArgs.IsPublic,
		Ingredients:  recipeArgs.Ingredients,
		Instructions: recipeArgs.Instructions,
	}

	dbRecipe, err := cfg.db.CreateRecipe(r.Context(), recipeParams)
	if err != nil {
		cfg.log.Warn("error creating recipe", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	var recipeDescription *string
	if dbRecipe.Description.Valid {
		recipeDescription = &dbRecipe.Description.String
	}

	recipeResponse := models.RecipeResponse{
		ID:              dbRecipe.ID,
		AuthorID:        dbRecipe.UserID.UUID,
		CreatedAt:       dbRecipe.CreatedAt,
		UpdatedAt:       dbRecipe.UpdatedAt,
		Title:           dbRecipe.Title,
		Description:     recipeDescription,
		IsPublic:        dbRecipe.IsPublic,
		Ingredients:     dbRecipe.Ingredients,
		Instructions:    dbRecipe.Instructions,
		Nutrition:       dbRecipe.Nutrition,
		NutritionStatus: dbRecipe.NutritionStatus,
	}

	cfg.respondWithJSON(w, http.StatusCreated, recipeResponse)
}
