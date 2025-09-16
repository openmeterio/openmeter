package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var PortalTokenIssuer = "openmeter"

type JTWPortalTokenClaims struct {
	jwt.RegisteredClaims

	// Namespace is the namespace where the token is created.
	Namespace string

	// Id is the unique identifier of the token.
	Id string

	// AllowedMeterSlugs is a list of meter slugs that the token allows access to.
	AllowedMeterSlugs []string
}

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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JTWPortalTokenClaims{
		Namespace: input.Namespace,
		Id:        id,
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
func (a *adapter) Validate(ctx context.Context, tokenString string) (*portal.PortalTokenClaims, error) {
	opts := []jwt.ParserOption{
		jwt.WithStrictDecoding(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(PortalTokenIssuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return a.secret, nil
	}

	jwtToken, err := jwt.ParseWithClaims(tokenString, &JTWPortalTokenClaims{}, keyFunc, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot parse claims: %w", err)
	}

	jwtClaims, ok := jwtToken.Claims.(*JTWPortalTokenClaims)
	if !ok {
		return nil, fmt.Errorf("not a portal token claims")
	}

	claims := &portal.PortalTokenClaims{
		Namespace:         jwtClaims.Namespace,
		Id:                jwtClaims.Id,
		AllowedMeterSlugs: jwtClaims.AllowedMeterSlugs,
		Subject:           jwtClaims.Subject,
	}

	if jwtClaims.ExpiresAt != nil {
		claims.ExpiresAt = &jwtClaims.ExpiresAt.Time
	}

	return claims, nil
}

// ListTokens lists portal tokens.
func (a *adapter) ListTokens(context.Context, portal.ListTokensInput) (pagination.Result[*portal.PortalToken], error) {
	var resp pagination.Result[*portal.PortalToken]

	return resp, models.NewGenericNotImplementedError(fmt.Errorf("listing tokens"))
}

// InvalidateToken invalidates a portal token.
func (a *adapter) InvalidateToken(ctx context.Context, input portal.InvalidateTokenInput) error {
	return models.NewGenericNotImplementedError(fmt.Errorf("invalidate tokens"))
}
