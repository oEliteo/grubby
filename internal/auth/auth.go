package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrNoAuthHeaderIncluded = errors.New("no auth header included in request")

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	currentTime := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    "grubby-access",
		IssuedAt:  jwt.NewNumericDate(currentTime),
		ExpiresAt: jwt.NewNumericDate(currentTime.Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, err
	}

	return uuid.Parse(subject)
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("token does not contain correct prefix")
	}

	trimmedToken := strings.TrimPrefix(authHeader, "Bearer ")
	trimmedToken = strings.TrimSpace(trimmedToken)

	if trimmedToken == "" {
		return "", errors.New("token is empty after trimming")
	}

	return trimmedToken, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}

	if !strings.HasPrefix(authHeader, "ApiKey ") {
		return "", errors.New("token does not contain correct prefix")
	}

	trimmedToken := strings.TrimPrefix(authHeader, "ApiKey ")
	trimmedToken = strings.TrimSpace(trimmedToken)

	if trimmedToken == "" {
		return "", errors.New("token is empty after trimming")
	}

	return trimmedToken, nil
}
