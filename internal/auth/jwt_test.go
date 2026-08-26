package auth

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestValidTokenCorrectSecret(t *testing.T) {
	secret := "testing-secret-shhhh"
	userID := uuid.New()
	expiry := time.Minute * 15
	signedTokenString, err := MakeJWT(userID, secret, expiry)

	if err != nil {
		t.Fatalf("Error creating JWT signed string")
	}

	resultID, err := ValidateJWT(signedTokenString, secret)

	if err != nil {
		t.Fatalf("Error validating token for some reason")
	}

	if resultID != userID {
		t.Errorf("expected %v, got %v", userID, resultID)
	}
}

func TestExpiredTokenCorrectSecret(t *testing.T) {
	secret := "super-duper-testing-secret-shhhh"
	userID := uuid.New()
	expiry := -time.Hour
	signedTokenString, err := MakeJWT(userID, secret, expiry)

	if err != nil {
		t.Fatalf("Error creating JWT signed string")
	}

	_, err = ValidateJWT(signedTokenString, secret)

	if err == nil {
		t.Errorf("expected error to not be nil, got %v", err)
	}
}

func TestValidTokenIncorrectSecret(t *testing.T) {
	secret := "the-ultimate-testing-secret-dont-tell-anybody"
	userID := uuid.New()
	expiry := time.Minute * 15
	signedTokenString, err := MakeJWT(userID, secret, expiry)

	if err != nil {
		t.Fatalf("error creating JWT signed string")
	}

	_, err = ValidateJWT(signedTokenString, "Secrets-Out")

	if err == nil {
		t.Errorf("expected error to not be nil, got %v", err)
	}
}
