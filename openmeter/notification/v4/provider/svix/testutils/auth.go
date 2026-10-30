package testutils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// NewSvixAuthToken signs a one-hour token for the shared Svix test organization.
func NewSvixAuthToken(signingSecret string) (string, error) {
	now := time.Now().UTC()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "svix-server",
		Subject:   "org_23rb8YdGqMT0qIzpgGwdXfHirMu",
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	})

	signedToken, err := token.SignedString([]byte(signingSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign Svix authentication token: %w", err)
	}

	return signedToken, nil
}
