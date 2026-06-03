# session

<!-- archie:ai-start -->

> Context-scoped authentication session carrier for the cloud/managed-hosting auth layer. Stores AuthenticationSession (org ID, role, permissions, user ID) in context under a typed key; used by auth middleware to propagate identity through request handlers.

## Patterns

**Typed context key to avoid collisions** — AuthenticatorContextKey is a named string type; AuthenticationSessionKey is of that type. Never use a plain string as a context key. (`type AuthenticatorContextKey string
const AuthenticationSessionKey AuthenticatorContextKey = "active_organization_id"`)
**GetActiveSession returns nil on missing or wrong type** — Uses a type assertion with ok-check and returns nil instead of panicking. Callers must nil-check before using the session. (`func GetActiveSession(ctx context.Context) *AuthenticationSession { if c, ok := ctx.Value(AuthenticationSessionKey).(*AuthenticationSession); ok { return c }; return nil }`)
**Validate on construction** — NewAuthenticationSession calls session.Validate() before returning; invalid sessions are rejected at construction. OrgRole OR OrgPermissions must be non-empty. (`session, err := NewAuthenticationSession(orgID, orgSlug, orgRole, userID, perms) // err if OrgID empty or both OrgRole and OrgPermissions empty`)
**WithLogger enriches slog with session fields** — AuthenticationSession.WithLogger(logger) adds orgId, userId, orgSlug, orgRole, orgPermissions as structured log fields. (`logger = session.WithLogger(logger)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `session.go` | Entire package: context key type, AuthenticationSession struct, constructor, validator, context getter helpers, logger enrichment. | Storing AuthenticationSession by value in context — the getter type-asserts to *AuthenticationSession, so a value type always returns nil from GetActiveSession. |

## Anti-Patterns

- Storing AuthenticationSession by value in context (must be pointer *AuthenticationSession)
- Using a plain string context key instead of the typed AuthenticatorContextKey constant
- Calling ctx.Value(AuthenticationSessionKey) directly outside of GetActiveSession — always use the helper
- Adding business logic (authorization checks, permission resolution) to this package — it is purely a session carrier

## Decisions

- **OrgRole OR OrgPermissions must be non-empty, not both required** — Different auth flows supply either a role string or a fine-grained permission list; requiring only one keeps the session compatible with both flows without two separate session types.

<!-- archie:ai-end -->
