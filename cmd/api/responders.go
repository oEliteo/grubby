package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func (cfg *apiConfig) respondWithError(w http.ResponseWriter, statusCode int, msg string) {
	type errType struct {
		Error string `json:"error"`
	}
	errStruct := errType{
		Error: msg,
	}
	cfg.respondWithJSON(w, statusCode, errStruct)
}

func (cfg *apiConfig) respondWithJSON(w http.ResponseWriter, statusCode int, payload any) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(dat)
}
