package portal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Service is the interface for the portal service.
type Service interface {
	PortalTokenService
}

// PortalTokenService is the service for the portal token management.
type PortalTokenService interface {
	CreateToken(ctx context.Context, input CreateTokenInput) (*PortalToken, error)
	Validate(ctx context.Context, tokenString string) (*PortalTokenClaims, error)
	ListTokens(ctx context.Context, input ListTokensInput) (pagination.Result[*PortalToken], error)
	InvalidateToken(ctx context.Context, input InvalidateTokenInput) error
}

// PortalTokenClaims is the claims struct for the portal token.
type PortalTokenClaims struct {
	// Namespace is the namespace where the token is created.
	Namespace string

	// Id is the unique identifier of the token.
	Id string

	// AllowedMeterSlugs is a list of meter slugs that the token allows access to.
	// All meter slugs are allowed if the list is empty.
	AllowedMeterSlugs []string

	// Subject is the subject of the token.
	Subject string

	// ExpiresAt is the expiration time of the token if any.
	ExpiresAt *time.Time
}

// GetAllowedMeterSlugs returns the list of allowed meter slugs.
func (c *PortalTokenClaims) GetAllowedMeterSlugs() ([]string, error) {
	return c.AllowedMeterSlugs, nil
}

// PortalToken is the struct for the portal token.
type PortalToken struct {
	Id                *string
	AllowedMeterSlugs *[]string
	ExpiresAt         *time.Time
	Subject           string
	// Only set when creating a token.
	Token *string
}

// CreateTokenInput is the input for the CreateToken method.
type CreateTokenInput struct {
	Namespace         string
	Subject           string
	AllowedMeterSlugs *[]string
	ExpiresAt         *time.Time
}

func (i CreateTokenInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if i.Subject == "" {
		errs = append(errs, fmt.Errorf("subject is required"))
	}

	if i.ExpiresAt != nil && i.ExpiresAt.Before(time.Now()) {
		errs = append(errs, fmt.Errorf("expiration date must be in the future"))
	}

	if i.AllowedMeterSlugs != nil {
		for _, slug := range *i.AllowedMeterSlugs {
			if slug == "" {
				errs = append(errs, fmt.Errorf("allowed meter slug cannot be empty"))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ListTokensInput is the input for the ListTokens method.
type ListTokensInput struct {
	Namespace string
	pagination.Page
}

func (i ListTokensInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := i.Page.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// InvalidateTokenInput is the input for the InvalidateToken method.
type InvalidateTokenInput struct {
	Namespace string
	ID        *string
	Subject   *string
}

func (i InvalidateTokenInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if i.ID == nil && i.Subject == nil {
		errs = append(errs, fmt.Errorf("either id or subject must be provided"))
	}

	if i.ID != nil && *i.ID == "" {
		errs = append(errs, fmt.Errorf("id cannot be empty"))
	}

	if i.Subject != nil && *i.Subject == "" {
		errs = append(errs, fmt.Errorf("subject cannot be empty"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
