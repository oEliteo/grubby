package main

import (
	"database/sql"
	"encoding/json"
	"grubby/internal/database"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// This file contains handler code for Create, Read, Update, and Delete operations on recipes.
type Ingredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

type RecipeArgs struct {
	ID           uuid.UUID    `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	IsPublic     bool         `json:"is_public"`
	Ingredients  []Ingredient `json:"ingredients"`
	Instructions []string     `json:"instructions"`
}

type RecipeResponse struct {
	ID              uuid.UUID       `json:"id"`
	AuthorID        uuid.UUID       `json:"author"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Title           string          `json:"title"`
	Description     *string         `json:"description,omitempty"`
	IsPublic        bool            `json:"is_public"`
	Ingredients     []Ingredient    `json:"ingredients"`
	Instructions    []string        `json:"instructions"`
	Nutrition       json.RawMessage `json:"nutrition,omitempty"`
	NutritionStatus string          `json:"nutrition_status"`
}

func NewNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}

	return sql.NullString{String: s, Valid: true}
}

func MarshalJSONB[T any](v T) (json.RawMessage, error) {
	return json.Marshal(v)
}

func UnmarshalJSONB[T any](data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}

func (cfg *apiConfig) handleRecipeCreate(w http.ResponseWriter, r *http.Request) {
	authUserID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		cfg.log.Warn("userID in context is not a uuid")
		cfg.respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	recipeArgs := RecipeArgs{}
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

	ingredientsJSON, err := MarshalJSONB(recipeArgs.Ingredients)
	if err != nil {
		cfg.log.Warn("error marshalling ingredients", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	instructionsJSON, err := MarshalJSONB(recipeArgs.Instructions)
	if err != nil {
		cfg.log.Warn("error marshalling instructions", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	if recipeArgs.Title == "" {
		cfg.log.Warn("recipe title is missing or empty")
		cfg.respondWithError(w, http.StatusBadRequest, "title is required")
		return
	}

	dbRecipeArgs := database.CreateRecipeParams{
		ID:           uuid.New(),
		UserID:       ownerID,
		Title:        recipeArgs.Title,
		Description:  NewNullString(recipeArgs.Description),
		IsPublic:     recipeArgs.IsPublic,
		Ingredients:  ingredientsJSON,
		Instructions: instructionsJSON,
	}

	dbRecipe, err := cfg.db.CreateRecipe(r.Context(), dbRecipeArgs)
	if err != nil {
		cfg.log.Warn("error creating recipe", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusBadRequest, "bad request")
		return
	}

	var descriptionPtr *string

	if dbRecipe.Description.Valid {
		descriptionPtr = &dbRecipe.Description.String
	}

	var nutritionData json.RawMessage
	if dbRecipe.Nutrition != nil {
		nutritionData = *dbRecipe.Nutrition
	}

	ingredients, err := UnmarshalJSONB[[]Ingredient](dbRecipe.Ingredients)
	if err != nil {
		cfg.log.Warn("error unmarshalling ingredients", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	instructions, err := UnmarshalJSONB[[]string](dbRecipe.Instructions)
	if err != nil {
		cfg.log.Warn("error unmarshalling instructions", slog.String("error", err.Error()))
		cfg.respondWithError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	recipe := RecipeResponse{
		ID:              dbRecipe.ID,
		AuthorID:        dbRecipe.UserID.UUID,
		CreatedAt:       dbRecipe.CreatedAt,
		UpdatedAt:       dbRecipe.UpdatedAt,
		Title:           dbRecipe.Title,
		Description:     descriptionPtr,
		Ingredients:     ingredients,
		Instructions:    instructions,
		NutritionStatus: dbRecipe.NutritionStatus,
		Nutrition:       nutritionData,
		IsPublic:        dbRecipe.IsPublic,
	}

	cfg.respondWithJSON(w, http.StatusCreated, recipe)
}
