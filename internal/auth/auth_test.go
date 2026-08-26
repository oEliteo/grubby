package auth

import (
	"testing"
)

func TestSimplePasswordCheck(t *testing.T) {
	pwd := "easy123."
	hash, err := HashPassword(pwd)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	result, err := CheckPasswordHash(pwd, hash)
	if err != nil {
		t.Fatalf("Failed to check password against hash: %v", err)
	}

	if !result {
		t.Errorf("expected %v, got %v", true, result)
	}
}

func TestWrongPasswordCheck(t *testing.T) {
	pwd := "easy123."
	hash, err := HashPassword(pwd)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	result, err := CheckPasswordHash("easy123", hash)
	if err != nil {
		t.Fatalf("Failed to check password against hash: %v", err)
	}

	if result {
		t.Errorf("expected %v, got %v", false, result)
	}
}
