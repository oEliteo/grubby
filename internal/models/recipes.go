package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

type Ingredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

type Instruction struct {
	Text string `json:"text"`
}

type Nutrition struct {
	Calories           int64   `json:"calories"`
	Protein            float64 `json:"protein"`
	TotalCarbohydrates float64 `json:"total_carbohydrates"`
	TotalFat           float64 `json:"total_fat"`
}

type RecipeArgs struct {
	ID           uuid.UUID     `json:"id"`
	Title        string        `json:"title"`
	Description  *string       `json:"description,omitempty"`
	IsPublic     bool          `json:"is_public"`
	Ingredients  []Ingredient  `json:"ingredients"`
	Instructions []Instruction `json:"instructions"`
}

type RecipeResponse struct {
	ID              uuid.UUID     `json:"id"`
	AuthorID        uuid.UUID     `json:"author"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Title           string        `json:"title"`
	Description     *string       `json:"description,omitempty"`
	IsPublic        bool          `json:"is_public"`
	Ingredients     []Ingredient  `json:"ingredients"`
	Instructions    []Instruction `json:"instructions"`
	Nutrition       *Nutrition    `json:"nutrition,omitempty"`
	NutritionStatus string        `json:"nutrition_status"`
}

type RecipeWithAuthor struct {
	ID              uuid.UUID       `json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Title           string          `json:"title"`
	Description     *string         `json:"description,omitempty"`
	IsPublic        bool            `json:"is_public"`
	Ingredients     []Ingredient    `json:"ingredients"`
	Instructions    []Instruction   `json:"instructions"`
	Nutrition       json.RawMessage `json:"nutrition,omitempty"`
	NutritionStatus string          `json:"nutrition_status"`
	AuthorName      string          `json:"author_name"`
}
