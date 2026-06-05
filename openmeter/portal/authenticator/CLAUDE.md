# authenticator

<!-- archie:ai-start -->

> HTTP middleware that authenticates incoming requests by interpreting OpenAPI 3 security requirements and verifying portal Bearer tokens via portal.Service, injecting the authenticated subject into request context.

## Patterns

**OpenAPI-driven security resolution** — Middleware reads the swagger doc per route: getSecurityRequirements falls back to swagger.Security but prefers the matched operation's Security via chi RoutePattern. (`path := swagger.Paths.Find(rctx.RoutePattern()); if op := path.GetOperation(r.Method); op != nil && op.Security != nil { security = op.Security }`)
**Try requirements in order, join errors** — validateSecurityRequirements returns nil on first satisfied requirement; if none pass it returns errors.Join of all attempts. (`for _, sr := range securityRequirements { r, err = a.validateSecurityRequirement(sr, w, r); if err == nil { return r, nil }; errs = append(errs, err) }; return r, errors.Join(errs...)`)
**Security scheme name -> authenticator func** — getAuthenticatorFunc maps a scheme name to a verifier; the only known scheme is derived from api.PortalTokenAuthScopes split on '.'. (`case strings.Split(string(api.PortalTokenAuthScopes), ".")[0]: return a.verifyPortalToken`)
**Subject stored under typed context key** — verifyPortalToken injects claims.Subject under AuthenticatorSubjectSessionKey; GetAuthenticatedSubject reads it (empty string = unauthenticated). (`r = r.WithContext(context.WithValue(r.Context(), AuthenticatorSubjectSessionKey, claims.Subject))`)
**Meter-slug scoping enforced at middleware** — verifyPortalToken reads chi URLParam "meterSlug" and rejects when claims.AllowedMeterSlugs is non-empty and does not contain it. (`if len(claims.AllowedMeterSlugs) != 0 && !slices.Contains(claims.AllowedMeterSlugs, meterSlug) { return r, errors.New("meter slug not allowed") }`)
**Problem-JSON error responses** — Auth failures respond 401, internal failures 500 via models.NewStatusProblem(...).Respond(w); internal errors also go through errorHandler.HandleContext. (`models.NewStatusProblem(r.Context(), err, http.StatusUnauthorized).Respond(w)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `authenticator.go` | Whole package: Authenticator struct (portal.Service + errorsx.Handler), NewAuthenticator, NewAuthenticatorMiddlewareFunc(swagger), verifyPortalToken, and the typed context-key helpers. | verifyPortalToken double-checks expiry/subject even though adapter.Validate already enforces them — keep both. Bearer header must be exactly two space-split parts with 'Bearer' prefix. Unknown security scheme names yield a 500 internal error, not a 401. |

## Anti-Patterns

- Bypassing OpenAPI security requirements (hardcoding auth on/off) instead of letting getSecurityRequirements drive it.
- Reading the subject from anywhere but GetAuthenticatedSubject / AuthenticatorSubjectSessionKey.
- Returning the request unwrapped after a successful auth (must carry the subject in context via r.WithContext).
- Treating an empty subject as authenticated — GetAuthenticatedSubject returns false for empty strings.

## Decisions

- **Authentication is generic over OpenAPI 3 security requirements rather than hardcoded per route.** — The generated swagger doc is the single source of truth for which endpoints require which scheme, so middleware reflects spec changes automatically.
- **Verifier resolves schemes by name from api.PortalTokenAuthScopes.** — Keeps the middleware aligned with generated API scope constants instead of literal scheme strings.

## Example: Verify a portal Bearer token and inject the subject

```
claims, err := a.portal.Validate(r.Context(), bearerToken)
if err != nil { return r, fmt.Errorf("invalid token: %w", err) }
if claims.Subject == "" { return r, errors.New("invalid subject") }
if len(claims.AllowedMeterSlugs) != 0 && !slices.Contains(claims.AllowedMeterSlugs, meterSlug) {
  return r, errors.New("meter slug not allowed")
}
r = r.WithContext(context.WithValue(r.Context(), AuthenticatorSubjectSessionKey, claims.Subject))
```

<!-- archie:ai-end -->
