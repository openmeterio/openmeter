package authenticator

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AuthenticatorContextKey string

const (
	AuthenticatorSubjectSessionKey AuthenticatorContextKey = "openmeter_subject"
)

// GetAuthenticatedSubject returns the authenticated subject from the context.
func GetAuthenticatedSubject(ctx context.Context) (string, bool) {
	if c, ok := ctx.Value(AuthenticatorSubjectSessionKey).(string); ok {
		if c == "" {
			return c, false
		}

		return c, true
	}

	return "", false
}

type Authenticator struct {
	portal       portal.Service
	errorHandler errorsx.Handler
}

func NewAuthenticator(portal portal.Service, errorHandler errorsx.Handler) Authenticator {
	return Authenticator{
		portal:       portal,
		errorHandler: errorHandler,
	}
}

// NewAuthenticatorMiddlewareFunc returns a middleware function that authenticates
// requests based on the OpenAPI 3 security requirements.
// TODO: support custom claims
func (a Authenticator) NewAuthenticatorMiddlewareFunc(swagger *openapi3.T) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sr, err := a.getSecurityRequirements(swagger, r)
			if err != nil {
				a.errorHandler.HandleContext(r.Context(), err)
				models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w)

				return
			}

			if sr == nil {
				next.ServeHTTP(w, r)
				return
			}

			rWithContext, err := a.validateSecurityRequirements(*sr, w, r)
			if err != nil {
				models.NewStatusProblem(r.Context(), err, http.StatusUnauthorized).Respond(w)

				return
			}

			next.ServeHTTP(w, rWithContext)
		})
	}
}

func (a Authenticator) verifyPortalToken(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	meterSlug := chi.URLParam(r, "meterSlug")

	authorizationHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorizationHeader == "" {
		return r, errors.New("missing authorization header")
	}

	h := strings.Split(authorizationHeader, " ")
	if len(h) != 2 || h[0] != "Bearer" {
		return r, errors.New("invalid authorization header")
	}

	bearerToken := h[1]

	if bearerToken == "" {
		return r, errors.New("bearer token cannot be empty")
	}

	claims, err := a.portal.Validate(r.Context(), bearerToken)
	if err != nil {
		return r, fmt.Errorf("invalid token: %w", err)
	}

	if claims.Subject == "" {
		return r, errors.New("invalid subject")
	}

	if len(claims.AllowedMeterSlugs) != 0 && !slices.Contains(claims.AllowedMeterSlugs, meterSlug) {
		return r, errors.New("meter slug not allowed")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return r, errors.New("token expired")
	}

	r = r.WithContext(context.WithValue(r.Context(), AuthenticatorSubjectSessionKey, claims.Subject))

	return r, nil
}

func (a Authenticator) getSecurityRequirements(swagger *openapi3.T, r *http.Request) (*openapi3.SecurityRequirements, error) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return nil, errors.New("missing route context")
	}

	// Use the global security requirements as default
	security := &swagger.Security
	pattern := rctx.RoutePattern()
	path := swagger.Paths.Find(pattern)
	if path != nil {
		operation := path.GetOperation(r.Method)
		if operation != nil && operation.Security != nil {
			security = operation.Security
		}
	}

	return security, nil
}

// validateSecurityRequirements goes through multiple OpenAPI 3 security
// requirements in order and returns nil on the first valid requirement.
// If no requirement is met, errors are returned in order.
func (a Authenticator) validateSecurityRequirements(securityRequirements openapi3.SecurityRequirements, w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	if len(securityRequirements) == 0 {
		return r, nil
	}

	errs := []error{}
	for _, sr := range securityRequirements {
		var err error
		r, err = a.validateSecurityRequirement(sr, w, r)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		return r, nil
	}

	return r, errors.Join(errs...)
}

// validateSecurityRequirement validates a single OpenAPI 3 security requirement
func (a Authenticator) validateSecurityRequirement(securityRequirement openapi3.SecurityRequirement, w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	if len(securityRequirement) == 0 {
		return r, nil
	}

	for name := range securityRequirement {
		f := a.getAuthenticatorFunc(name)
		if f == nil {
			return r, fmt.Errorf("unknown security requirement: %s", name)
		}

		var err error
		r, err = f(w, r)
		if err != nil {
			return r, err
		}
	}

	return r, nil
}

func (a Authenticator) getAuthenticatorFunc(securitySchemaName string) func(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	switch securitySchemaName {
	case strings.Split(api.PortalTokenAuthScopes, ".")[0]:
		return a.verifyPortalToken
	default:
		return nil
	}
}
