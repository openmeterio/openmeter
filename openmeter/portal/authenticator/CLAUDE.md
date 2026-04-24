# authenticator

<!-- archie:ai-start -->

> Chi middleware that authenticates HTTP requests against the v1 OpenAPI security requirements, dispatching to verifyPortalToken for PortalTokenAuth-scoped routes and injecting the validated subject into context.

## Patterns

**OpenAPI-driven security dispatch** — getSecurityRequirements reads per-operation security from the swagger spec (falling back to global); validateSecurityRequirements iterates alternatives and returns on first success — mirrors OpenAPI OR semantics. (`for _, sr := range securityRequirements { r, err = a.validateSecurityRequirement(sr, w, r); if err == nil { return r, nil } }`)
**Subject injected via typed context key** — Authenticated subject is stored under AuthenticatorSubjectSessionKey (typed AuthenticatorContextKey string). Retrieve with GetAuthenticatedSubject(ctx) — never use raw string key. (`r = r.WithContext(context.WithValue(r.Context(), AuthenticatorSubjectSessionKey, claims.Subject))`)
**Security scheme name matched by prefix split** — getAuthenticatorFunc splits api.PortalTokenAuthScopes on '.' and matches by first segment — adding a new auth scheme requires a new case here. (`case strings.Split(string(api.PortalTokenAuthScopes), ".")[0]: return a.verifyPortalToken`)
**Nil security requirement means unauthenticated route** — If getSecurityRequirements returns nil (no matching route/operation), the middleware calls next directly without authentication. (`if sr == nil { next.ServeHTTP(w, r); return }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `authenticator.go` | Entire authenticator: struct, constructor, Chi middleware factory, portal token verification, OpenAPI security resolution | verifyPortalToken extracts meterSlug from chi.URLParam — if the route param name changes this check silently stops enforcing slug allowlists. AllowedMeterSlugs check uses slices.Contains; empty slice means all slugs allowed. |

## Anti-Patterns

- Adding new security schemes by hardcoding names as plain strings instead of using api.* constants split on '.'
- Calling portal.Validate outside verifyPortalToken — token validation must flow through the middleware, not ad hoc
- Skipping the AllowedMeterSlugs check when meterSlug is empty or route has no slug param
- Returning 200 or 500 for auth failures — unauthorized must always be 401 via models.NewStatusProblem

## Decisions

- **Security requirements read from swagger spec at request time rather than compiled into route handlers** — Keeps auth policy co-located with the OpenAPI spec; new endpoints automatically inherit security without changing middleware code.

<!-- archie:ai-end -->
