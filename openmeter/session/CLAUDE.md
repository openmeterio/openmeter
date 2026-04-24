# session

<!-- archie:ai-start -->

> Provides context-scoped authentication session storage and retrieval for the cloud/managed-hosting auth layer. Stores AuthenticationSession (org ID, role, permissions, user ID) in context under a typed key; used by auth middleware to propagate identity through request handlers.

## Patterns

**Typed context key to avoid collisions** — AuthenticatorContextKey is a named string type; the constant AuthenticationSessionKey is of that type. Never use a plain string as a context key in this package. (`type AuthenticatorContextKey string
const AuthenticationSessionKey AuthenticatorContextKey = "active_organization_id"`)
**GetActiveSession returns nil on missing/wrong type** — Use a type assertion with ok-check (`ctx.Value(key).(*AuthenticationSession)`) and return nil instead of panicking. Callers must nil-check before using the session. (`if c, ok := ctx.Value(AuthenticationSessionKey).(*AuthenticationSession); ok { return c }
return nil`)
**Validate on construction** — NewAuthenticationSession calls session.Validate() before returning; invalid sessions are rejected at construction time, not at read time. (`session, err := NewAuthenticationSession(orgID, orgSlug, orgRole, userID, perms)
// err non-nil if OrgID empty or both OrgRole and OrgPermissions empty`)
**WithLogger enriches slog with session fields** — Use AuthenticationSession.WithLogger(logger) to add orgId, userId, orgSlug, orgRole, orgPermissions as structured log fields instead of repeating slog.String(...) calls. (`logger = session.WithLogger(logger)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `session.go` | Single file containing the entire package: context key, AuthenticationSession struct, constructor, validator, context getter helpers, and logger enrichment. | Storing the session by value (not pointer) in context — the getter type-asserts to *AuthenticationSession, so storing a value type will always return nil. |

## Anti-Patterns

- Storing AuthenticationSession by value in context (must be pointer *AuthenticationSession)
- Using a plain string context key instead of the typed AuthenticatorContextKey constant
- Calling ctx.Value(AuthenticationSessionKey) outside of GetActiveSession — always use the helper
- Adding business logic (authorization checks, permission resolution) to this package — it is purely a session carrier

## Decisions

- **OrgRole OR OrgPermissions must be non-empty, not both required** — Different auth flows supply either a role string or a fine-grained permission list; requiring only one of them keeps the session compatible with both.

<!-- archie:ai-end -->
