package authenticator

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var PortalTokenIssuer = "openmeter"

// PortalTokenClaims is the claims struct for the portal token.
type PortalTokenClaims struct {
	jwt.RegisteredClaims

	// AllowedMeterSlugs is a list of meter slugs that the token allows access to.
	AllowedMeterSlugs []string `json:"allowed_meter_slugs,omitempty"`
}

// GetAllowedMeterSlugs returns the list of allowed meter slugs.
func (c *PortalTokenClaims) GetAllowedMeterSlugs() ([]string, error) {
	return c.AllowedMeterSlugs, nil
}

type PortalToken struct {
	AllowedMeterSlugs *[]string  `json:"allowedMeterSlugs,omitempty"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
	Subject           string     `json:"subject"`
	Token             *string    `json:"token,omitempty"`
}

type PortalTokenStrategy struct {
	secret []byte
	expire time.Duration
}

func NewPortalTokenStrategy(secret string, expire time.Duration) (*PortalTokenStrategy, error) {
	if secret == "" {
		return nil, fmt.Errorf("token secret is required")
	}

	if expire.Seconds() == 0 {
		return nil, fmt.Errorf("token duration is required")
	}

	return &PortalTokenStrategy{
		secret: []byte(secret),
		expire: expire,
	}, nil
}

func (t *PortalTokenStrategy) Generate(subject string, allowedMeterSlugs *[]string, expiresAt *time.Time) (*PortalToken, error) {
	// set the default expiration time
	if expiresAt == nil {
		e := time.Now().Add(t.expire)
		expiresAt = &e
	}
	if allowedMeterSlugs == nil {
		allowedMeterSlugs = &[]string{}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		PortalTokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   subject,
				ExpiresAt: jwt.NewNumericDate(*expiresAt),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    PortalTokenIssuer,
			},
			AllowedMeterSlugs: *allowedMeterSlugs,
		})

	tokenString, err := token.SignedString(t.secret)
	if err != nil {
		return nil, err
	}

	return &PortalToken{
		AllowedMeterSlugs: allowedMeterSlugs,
		ExpiresAt:         expiresAt,
		Subject:           subject,
		Token:             &tokenString,
	}, nil
}

func (t *PortalTokenStrategy) Validate(tokenString string) (*PortalTokenClaims, error) {
	opts := []jwt.ParserOption{
		jwt.WithStrictDecoding(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(PortalTokenIssuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	}
	token, err := jwt.ParseWithClaims(tokenString, &PortalTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return t.secret, nil
	}, opts...)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*PortalTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
