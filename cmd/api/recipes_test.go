package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func TestHandleRecipeCreateUnauthorized(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	recipeArgs := RecipeArgs{
		Title:       "Test Recipe",
		Description: "A delicious test recipe",
		IsPublic:    true,
		Ingredients: []Ingredient{
			{Name: "Flour", Amount: 2.0, Unit: "cups"},
		},
		Instructions: []string{"Mix ingredients", "Bake at 350F"},
	}

	data, err := json.Marshal(recipeArgs)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	// No Authorization header provided, simulating missing/bad auth context

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %v got %v", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandleRecipeCreateBadRequest(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	badData := []byte(`{"title": 12345}`)

	reader := bytes.NewReader(badData)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %v got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleRecipeCreateSuccess(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	recipeArgs := RecipeArgs{
		Title:       "Chocolate Chip Cookies",
		Description: "Classic homemade cookies",
		IsPublic:    true,
		Ingredients: []Ingredient{
			{Name: "Sugar", Amount: 1.0, Unit: "cup"},
			{Name: "Butter", Amount: 0.5, Unit: "cup"},
		},
		Instructions: []string{
			"Cream butter and sugar.",
			"Bake at 375 degrees for 10 minutes.",
		},
	}

	data, err := json.Marshal(recipeArgs)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %v got %v (body: %s)", http.StatusCreated, rr.Code, rr.Body.String())
	}

	returnedRecipe := RecipeResponse{}
	if err := json.Unmarshal(rr.Body.Bytes(), &returnedRecipe); err != nil {
		t.Fatalf("error unmarshalling response body: %v\n", err)
	}

	if returnedRecipe.ID == uuid.Nil {
		t.Fatalf("expected a valid recipe ID, got nil uuid")
	}

	if returnedRecipe.Title != recipeArgs.Title {
		t.Fatalf("expected title %v got %v", recipeArgs.Title, returnedRecipe.Title)
	}

	if returnedRecipe.AuthorID != newUsr.UserResponse.ID {
		t.Fatalf("expected author ID %v got %v", newUsr.UserResponse.ID, returnedRecipe.AuthorID)
	}

	if len(returnedRecipe.Ingredients) != 2 {
		t.Fatalf("expected 2 ingredients got %d", len(returnedRecipe.Ingredients))
	}

	if len(returnedRecipe.Instructions) != 2 {
		t.Fatalf("expected 2 instructions got %d", len(returnedRecipe.Instructions))
	}
}

func TestHandleRecipeCreateEmptyDescription(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	recipeArgs := RecipeArgs{
		Title:       "Simple Toast",
		Description: "",
		IsPublic:    false,
		Ingredients: []Ingredient{
			{Name: "Bread", Amount: 1.0, Unit: "slice"},
		},
		Instructions: []string{"Toast the bread."},
	}

	data, err := json.Marshal(recipeArgs)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %v got %v (body: %s)", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var rawResp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &rawResp); err != nil {
		t.Fatalf("error unmarshalling into map: %v", err)
	}

	if _, exists := rawResp["description"]; exists {
		t.Fatalf("expected description field to be omitted, but it was present with value: %v", rawResp["description"])
	}
}

func TestHandleRecipeCreateMissingTitle(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	recipeArgs := RecipeArgs{
		Title:        "",
		Ingredients:  []Ingredient{{Name: "Water", Amount: 1.0, Unit: "cup"}},
		Instructions: []string{"Drink it."},
	}

	data, err := json.Marshal(recipeArgs)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %v got %v (body: %s)", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestHandleRecipeCreateTitleTooLong(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	// Generate a title exceeding the 150-character VARCHAR limit
	longTitle := ""
	for range 160 {
		longTitle += "a"
	}

	recipeArgs := RecipeArgs{
		Title:       longTitle,
		Description: "A valid description",
		IsPublic:    true,
		Ingredients: []Ingredient{
			{Name: "Flour", Amount: 1.0, Unit: "cup"},
		},
		Instructions: []string{"Bake"},
	}

	data, err := json.Marshal(recipeArgs)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %v got %v (body: %s)", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestHandleRecipeCreateEmptyIngredients(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	recipeArgs := RecipeArgs{
		Title:        "Valid Title",
		Description:  "A valid description",
		IsPublic:     true,
		Ingredients:  []Ingredient{},
		Instructions: []string{"Bake"},
	}

	data, err := json.Marshal(recipeArgs)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %v got %v (body: %s)", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestHandleRecipeCreateTooManyIngredients(t *testing.T) {
	cfg, db, err := newTestConfig()
	if err != nil {
		t.Fatalf("error initializing test apiConfig: %v\n", err)
	}
	defer db.Close()
	defer resetTestDB(t, db)

	newUsr := createNewUser(t, db, cfg)

	oversizedIngredients := make([]Ingredient, 51)
	for i := range 51 {
		oversizedIngredients[i] = Ingredient{Name: "Ingredient", Amount: 1.0, Unit: "g"}
	}

	recipeArgs := RecipeArgs{
		Title:        "Valid Title",
		Description:  "A valid description",
		IsPublic:     true,
		Ingredients:  oversizedIngredients,
		Instructions: []string{"Bake"},
	}

	data, err := json.Marshal(recipeArgs)
	if err != nil {
		t.Fatalf("error marshalling payload: %v\n", err)
	}

	reader := bytes.NewReader(data)
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", reader)
	request.Header.Set("Authorization", "Bearer "+newUsr.Token)

	rr := httptest.NewRecorder()
	handler := cfg.Authenticate(cfg.handleRecipeCreate)
	handler(rr, request)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %v got %v (body: %s)", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}
