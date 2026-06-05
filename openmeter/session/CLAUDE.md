# session

<!-- archie:ai-start -->

> Defines the request-scoped AuthenticationSession (org/user identity + permissions) and context helpers to store/retrieve it. It is the single source of authenticated identity consumed across app, customer, meter, subscription and productcatalog packages.

## Patterns

**Context-keyed session storage** — Session is stored/retrieved via a typed context key (AuthenticatorContextKey) using GetActiveSession with a type-asserted ctx.Value lookup that returns nil when absent. (`if c, ok := ctx.Value(AuthenticationSessionKey).(*AuthenticationSession); ok { return c }`)
**Validated constructor** — NewAuthenticationSession builds the struct then calls Validate() and returns an error rather than an invalid session; OrgID and (OrgRole or OrgPermissions) are required. (`if err := session.Validate(); err != nil { return nil, fmt.Errorf(...) }`)
**errors.Join validation** — Validate() collects into `var errs []error` and returns errors.Join(errs...), matching the repo-wide validation convention. (`errs = append(errs, errors.New("orgID is required"))`)
**Nil-safe accessors** — GetSessionUserID returns *string and yields nil for missing session or empty UserID, so callers can distinguish absence from empty. (`GetSessionUserID(ctx) returns nil when no active session`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `session.go` | Entire package: AuthenticationSession struct, context key, GetActiveSession/GetSessionUserID accessors, NewAuthenticationSession constructor, Validate(), and WithLogger. | AuthenticationSessionKey is the constant string "active_organization_id" typed as AuthenticatorContextKey — use the typed key, never a raw string, or the type assertion in GetActiveSession will miss it. |

## Anti-Patterns

- Constructing AuthenticationSession literals directly without Validate() (bypasses required-field checks).
- Storing/reading the session under a plain string context key instead of AuthenticationSessionKey.
- Assuming GetActiveSession never returns nil — callers must nil-check.

## Decisions

- **Identity is org-centric with permission list plus role.** — Validate accepts either OrgRole or OrgPermissions, supporting both role-based and explicit-permission auth models in a multi-tenant system.

## Example: Read the active org/user from context

```
import "github.com/openmeterio/openmeter/openmeter/session"

if s := session.GetActiveSession(ctx); s != nil {
	log = s.WithLogger(log)
	_ = s.OrgID
}
```

<!-- archie:ai-end -->
