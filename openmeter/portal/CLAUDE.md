# portal

<!-- archie:ai-start -->

> Issues and validates short-lived HS256 JWT portal tokens scoped to namespace, subject, and an optional meter slug allowlist; ListTokens and InvalidateToken are intentionally unimplemented stubs. The root package owns the Service interface and all input/output types with Validate() methods; adapter/ and authenticator/ and httphandler/ sub-packages provide the JWT implementation, Chi middleware, and HTTP handlers.

## Patterns

**Input Validate() before service call** — CreateTokenInput, ListTokensInput, and InvalidateTokenInput each expose Validate() with errors.Join. Adapters and httphandlers must call Validate() before delegating to portal.Service. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Token field set only in CreateToken response path** — PortalToken.Token is *string and nil in all read paths. The JWT string is assigned only after token creation — never inside toAPIPortalToken mapping, which is shared by list/read paths. (`token, err := adapter.CreateToken(ctx, input); // then: token.Token = &jwtString`)
**ListTokens/InvalidateToken return GenericNotImplementedError** — These methods are intentionally unimplemented on the JWT adapter (stateless by design). A separate noop.go provides a full stub for test wiring that also returns NotImplementedError. (`func (a *Adapter) ListTokens(...) (pagination.Result[*PortalToken], error) { return pagination.Result[*PortalToken]{}, models.NewGenericNotImplementedError("list portal tokens") }`)
**Meter slug allowlist validation in httphandler, not portal.Service** — CreateToken httphandler validates AllowedMeterSlugs against meter.Service before calling portal.Service.CreateToken. portal.Service has no dependency on meter.Service — cross-domain validation lives at the transport boundary. (`// In httphandler/portal.go operation func: for _, slug := range slugs { _, err := meterSvc.GetMeterByIDOrSlug(ctx, ns, slug) }`)
**AllowedMeterSlugs nil vs empty semantics** — AllowedMeterSlugs *[]string: nil means all meter slugs allowed; non-nil empty slice means no slugs allowed. Callers and the authenticator must distinguish these two cases. (`if input.AllowedMeterSlugs != nil { for _, slug := range *input.AllowedMeterSlugs { if slug == "" { errs = append(errs, ...) } } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service (composed of PortalTokenService), PortalToken, PortalTokenClaims, CreateTokenInput, ListTokensInput, InvalidateTokenInput, and all Validate() methods. | AllowedMeterSlugs is *[]string — nil means unrestricted; non-nil empty means fully restricted. Callers must not conflate the two cases. |
| `adapter/adapter.go` | JWT-backed stateless adapter. Config.SharedSecret must be non-empty. Strict JWT parse options (WithExpirationRequired, WithStrictDecoding, WithValidMethods, WithIssuer) must not be relaxed. | Never store token state in DB here — stateless by design. A DB-backed impl would be a separate adapter type. |
| `authenticator/authenticator.go` | Chi middleware dispatching per OpenAPI security scheme; injects validated PortalTokenClaims subject into context via typed context key. | Auth failures must always be 401 via models.NewStatusProblem — never 200 or 500. AllowedMeterSlugs check must happen even when meterSlug param is absent from route. |
| `httphandler/mapping.go` | API↔domain type conversions. toAPIPortalToken must not set the Token field — that is set only in the CreateToken operation. | Inlining conversions in portal.go breaks the mapping.go convention. Token field leaking into non-create responses exposes JWT strings incorrectly. |

## Anti-Patterns

- Relaxing JWT validation options (removing WithExpirationRequired, WithStrictDecoding, WithValidMethods) in the adapter
- Setting PortalToken.Token inside toAPIPortalToken — must only be assigned in the CreateToken response path
- Adding DB persistence to the JWT adapter — stateless by design; a DB-backed impl is a separate adapter
- Skipping AllowedMeterSlugs validation in httphandler when meterSlug param is absent from route
- Returning 500 for auth failures in the authenticator middleware — must always be 401

## Decisions

- **Stateless JWT tokens with no DB persistence for ListTokens/InvalidateToken** — Portal tokens are short-lived and scoped; avoiding a DB table removes a write-path dependency and keeps token issuance fast. ListTokens/InvalidateToken are explicitly unimplemented until a DB-backed adapter is needed.
- **Meter slug allowlist validation in httphandler, not portal.Service** — portal.Service has no import dependency on meter.Service — cross-domain meter existence validation is a transport-boundary concern, preserving the clean domain separation.

## Example: CreateToken handler — meter slug validation before portal.Service call

```
// In httphandler/portal.go operation func:
if input.AllowedMeterSlugs != nil {
    for _, slug := range *input.AllowedMeterSlugs {
        if _, err := h.meterSvc.GetMeterByIDOrSlug(ctx, ns, slug); err != nil {
            return nil, models.NewGenericValidationError(fmt.Errorf("meter slug %q not found", slug))
        }
    }
}
token, err := h.portalSvc.CreateToken(ctx, input)
if err != nil {
    return nil, err
}
// Token field is set here — NOT inside toAPIPortalToken
apiToken := toAPIPortalToken(token)
apiToken.Token = token.Token
```

<!-- archie:ai-end -->
