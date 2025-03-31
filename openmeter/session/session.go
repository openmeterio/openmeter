package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

type AuthenticatorContextKey string

const (
	AuthenticationSessionKey AuthenticatorContextKey = "active_organization_id"
)

// GetUserID returns the user ID from the session if any
func GetSessionUserID(ctx context.Context) *string {
	if session := GetActiveSession(ctx); session != nil {
		if session.UserID == "" {
			return nil
		}

		return &session.UserID
	}

	return nil
}

// GetActiveSession returns the active session from the context if it exists
func GetActiveSession(ctx context.Context) *AuthenticationSession {
	if c, ok := ctx.Value(AuthenticationSessionKey).(*AuthenticationSession); ok {
		return c
	}

	return nil
}

// NewAuthenticationSession creates a new authentication session
func NewAuthenticationSession(orgID, orgSlug, orgRole, userID string, orgPermissions []string) (*AuthenticationSession, error) {
	session := &AuthenticationSession{
		UserID:         userID,
		OrgID:          orgID,
		OrgSlug:        orgSlug,
		OrgRole:        orgRole,
		OrgPermissions: orgPermissions,
	}

	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}

	return session, nil
}

// AuthenticationSession represents the authentication session for a user
type AuthenticationSession struct {
	UserID         string
	OrgID          string
	OrgSlug        string
	OrgRole        string
	OrgPermissions []string
}

// Validate validates the session.
func (s AuthenticationSession) Validate() error {
	var errs []error

	if s.OrgID == "" {
		errs = append(errs, errors.New("orgID is required"))
	}

	if s.OrgRole == "" && len(s.OrgPermissions) == 0 {
		errs = append(errs, errors.New("orgRole or orgPermissions is required"))
	}

	return errors.Join(errs...)
}

// WithLogger returns a new logger with the session context
func (s AuthenticationSession) WithLogger(logger *slog.Logger) *slog.Logger {
	return logger.With(
		slog.String("orgId", s.OrgID),
		slog.String("userId", s.UserID),
		slog.String("orgSlug", s.OrgSlug),
		slog.String("orgRole", s.OrgRole),
		slog.String("orgPermissions", strings.Join(s.OrgPermissions, ",")),
	)
}
