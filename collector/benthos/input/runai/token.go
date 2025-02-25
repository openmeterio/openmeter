package runai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenRequest represents the payload sent to obtain an API token.
type TokenRequest struct {
	GrantType string `json:"grantType"`
	AppId     string `json:"AppId"`
	AppSecret string `json:"AppSecret"`
}

// TokenResponse represents the response containing the access token.
type TokenResponse struct {
	AccessToken string `json:"accessToken"`
}

// RefreshToken refreshes the token if it's expired.
func (s *Service) RefreshToken(ctx context.Context) error {
	// If the token is not set, refresh it.
	if s.token == "" {
		token, err := s.NewToken(ctx)
		if err != nil {
			return err
		}

		s.SetToken(token)
	}

	// If the token is expired, refresh it.
	err := s.verifyToken()
	if err != nil {
		token, err := s.NewToken(ctx)
		if err != nil {
			return err
		}

		s.SetToken(token)
	}

	return nil
}

// GetToken gets the current token for the service.
func (s *Service) GetToken() string {
	return s.token
}

// SetToken sets the token for the service.
func (s *Service) SetToken(token string) {
	s.token = token
}

// NewToken gets an access token for the application.
func (s *Service) NewToken(ctx context.Context) (string, error) {
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetBody(TokenRequest{
			GrantType: "app_token",
			AppId:     s.appID,
			AppSecret: s.appSecret,
		}).
		SetResult(&TokenResponse{}).
		Post("/api/v1/token")
	if err != nil {
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("failed to refresh token, status code: %d", resp.StatusCode())
	}

	result := resp.Result().(*TokenResponse)
	return result.AccessToken, nil
}

// verifyToken parses the token and checks if the token is expired.
func (s *Service) verifyToken() error {
	// Parse the token without verifying its signature.
	parser := jwt.NewParser(jwt.WithoutClaimsValidation(), jwt.WithExpirationRequired())
	t, _, err := parser.ParseUnverified(s.token, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	// Get the expiration time from the token.
	expirationTime, err := t.Claims.GetExpirationTime()
	if err != nil || expirationTime == nil {
		return fmt.Errorf("failed to get expiration time: %w", err)
	}

	// Check if the token has expired.
	if expirationTime.Before(time.Now()) {
		return fmt.Errorf("token has expired")
	}

	return nil
}
