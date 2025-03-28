package session

import "context"

type AuthenticatorContextKey string

const (
	AuthenticationSessionKey AuthenticatorContextKey = "active_organization_id"
)

type AuthenticationSession struct {
	UserID string
}

// GetUserID returns the user ID from the session if any
func GetSessionUserID(ctx context.Context) *string {
	s, ok := getActiveSession(ctx)
	if !ok {
		return nil
	}

	return &s.UserID
}

// getActiveSession returns the active session from the context if it exists
func getActiveSession(ctx context.Context) (*AuthenticationSession, bool) {
	if c, ok := ctx.Value(AuthenticationSessionKey).(*AuthenticationSession); ok {
		return c, true
	}

	return nil, false
}
