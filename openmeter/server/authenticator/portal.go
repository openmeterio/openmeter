// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authenticator

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var PortalTokenIssuer = "openmeter"

// PortalTokenClaims is the claims struct for the portal token.
type PortalTokenClaims struct {
	jwt.RegisteredClaims

	// Id is the unique identifier of the token.
	Id string `json:"id"`

	// AllowedMeterSlugs is a list of meter slugs that the token allows access to.
	AllowedMeterSlugs []string `json:"allowed_meter_slugs,omitempty"`
}

// GetAllowedMeterSlugs returns the list of allowed meter slugs.
func (c *PortalTokenClaims) GetAllowedMeterSlugs() ([]string, error) {
	return c.AllowedMeterSlugs, nil
}

type PortalToken struct {
	Id                *string    `json:"id"`
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
	id := uuid.New().String()

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
			Id: id,
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
		Id:                &id,
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
