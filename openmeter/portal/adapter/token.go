package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var PortalTokenIssuer = "openmeter"

// CreateToken creates a new portal token.
func (a *adapter) CreateToken(ctx context.Context, input portal.CreateTokenInput) (*portal.PortalToken, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	expiresAt := time.Now().Add(a.expire)
	allowedMeterSlugs := &[]string{}

	if input.ExpiresAt != nil {
		expiresAt = *input.ExpiresAt
	}

	if input.AllowedMeterSlugs != nil {
		allowedMeterSlugs = input.AllowedMeterSlugs
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, portal.PortalTokenClaims{
		Id: id,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   input.Subject,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    PortalTokenIssuer,
		},
		AllowedMeterSlugs: *allowedMeterSlugs,
	})

	tokenString, err := token.SignedString(a.secret)
	if err != nil {
		return nil, err
	}

	return &portal.PortalToken{
		Id:                &id,
		AllowedMeterSlugs: allowedMeterSlugs,
		ExpiresAt:         &expiresAt,
		Subject:           input.Subject,
		Token:             &tokenString,
	}, nil
}

// Validate validates a portal token.
func (a *adapter) Validate(tokenString string) (*portal.PortalTokenClaims, error) {
	opts := []jwt.ParserOption{
		jwt.WithStrictDecoding(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(PortalTokenIssuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return a.secret, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &portal.PortalTokenClaims{}, keyFunc, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot parse claims: %w", err)
	}

	claims, ok := token.Claims.(*portal.PortalTokenClaims)
	if !ok {
		return nil, fmt.Errorf("not a portal token claims")
	}

	return claims, nil
}

// ListTokens lists portal tokens.
func (a *adapter) ListTokens(context.Context, portal.ListTokensInput) (pagination.PagedResponse[*portal.PortalToken], error) {
	var resp pagination.PagedResponse[*portal.PortalToken]

	return resp, fmt.Errorf("not implemented")
}

// InvalidateToken invalidates a portal token.
func (a *adapter) InvalidateToken(ctx context.Context, input portal.InvalidateTokenInput) error {
	return fmt.Errorf("not implemented")
}
