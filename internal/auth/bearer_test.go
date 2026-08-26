package auth

import (
	"net/http"
	"testing"
)

func TestGetBearerTokenFromCorrectHeader(t *testing.T) {
	header := http.Header{}
	expected := "ThisIsABearerTokenTest"
	header.Set("Authorization", "Bearer "+expected)
	result, err := GetBearerToken(header)
	if err != nil {
		t.Errorf("expected %v, got %v", expected, result)
	}

	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestGetBearerTokenNoPrefix(t *testing.T) {
	header := http.Header{}
	expected := "tokennoprefix"
	header.Set("Authorization", expected)
	_, err := GetBearerToken(header)
	if err == nil {
		t.Errorf("expected error to be not nil, got %v", err)
	}
}

func TestGetBearerTokenEmptyAfterTrim(t *testing.T) {
	header := http.Header{}
	tmp := "Bearer "
	header.Set("Authorization", tmp)
	_, err := GetBearerToken(header)
	if err == nil {
		t.Errorf("expected error to be not nil, got %v", err)
	}
}
