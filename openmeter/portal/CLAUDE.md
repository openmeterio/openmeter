# portal

<!-- archie:ai-start -->

> Issues and validates short-lived HS256 JWT portal tokens scoped to namespace, subject, and an optional meter slug allowlist; ListTokens/InvalidateToken are intentionally unimplemented stubs. The root package owns the Service interface and all input/output types with Validate(); adapter/, authenticator/, and httphandler/ provide the JWT implementation, Chi middleware, and HTTP handlers.

## Patterns

**Input Validate() before service call** — CreateTokenInput, ListTokensInput, InvalidateTokenInput each expose Validate() with errors.Join; adapters and httphandlers call Validate() before delegating to portal.Service. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Token field set only in the CreateToken response path** — PortalToken.Token is *string and nil in all read paths; the JWT string is assigned only after token creation, never inside the shared toAPIPortalToken mapping. (`token, err := adapter.CreateToken(ctx, input); // then: token.Token = &jwtString`)
**ListTokens/InvalidateToken return GenericNotImplementedError** — These are intentionally unimplemented on the stateless JWT adapter (and the noop), returning NotImplementedError. (`return pagination.Result[*PortalToken]{}, models.NewGenericNotImplementedError("list portal tokens")`)
**Meter slug allowlist validation in httphandler, not portal.Service** — CreateToken httphandler validates AllowedMeterSlugs against meter.Service before calling portal.Service; portal.Service has no dependency on meter.Service. (`for _, slug := range slugs { _, err := meterSvc.GetMeterByIDOrSlug(ctx, ns, slug) }`)
**AllowedMeterSlugs nil vs empty semantics** — AllowedMeterSlugs *[]string: nil means all slugs allowed; non-nil empty means none allowed. Callers and the authenticator must distinguish these. (`if input.AllowedMeterSlugs != nil { for _, slug := range *input.AllowedMeterSlugs { if slug == "" { errs = append(errs, ...) } } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service (= PortalTokenService), PortalToken, PortalTokenClaims, CreateTokenInput, ListTokensInput, InvalidateTokenInput, and all Validate() methods. | AllowedMeterSlugs is *[]string — nil unrestricted, non-nil empty fully restricted; callers must not conflate the two cases. |
| `adapter/adapter.go` | JWT-backed stateless adapter; Config.SharedSecret must be non-empty; strict JWT parse options (WithExpirationRequired, WithStrictDecoding, WithValidMethods, WithIssuer). | Never store token state in a DB here — stateless by design; a DB-backed impl would be a separate adapter type. |
| `authenticator/authenticator.go` | Chi middleware dispatching per OpenAPI security scheme; injects validated PortalTokenClaims subject into context via a typed key. | Auth failures must always be 401 via models.NewStatusProblem — never 200 or 500; the AllowedMeterSlugs check must happen even when the meterSlug route param is absent. |
| `httphandler/mapping.go` | API↔domain conversions; toAPIPortalToken must not set the Token field — that is set only in the CreateToken operation. | Token field leaking into non-create responses exposes JWT strings incorrectly. |

## Anti-Patterns

- Relaxing JWT validation options (removing WithExpirationRequired, WithStrictDecoding, WithValidMethods) in the adapter
- Setting PortalToken.Token inside toAPIPortalToken — assign only in the CreateToken response path
- Adding DB persistence to the JWT adapter — stateless by design; a DB-backed impl is a separate adapter
- Skipping AllowedMeterSlugs validation in httphandler when the meterSlug param is absent from the route
- Returning 500 for auth failures in the authenticator middleware — must always be 401

## Decisions

- **Stateless JWT tokens with no DB persistence for ListTokens/InvalidateToken** — Portal tokens are short-lived and scoped; avoiding a DB table removes a write-path dependency and keeps issuance fast.
- **Meter slug allowlist validation in httphandler, not portal.Service** — portal.Service has no import dependency on meter.Service; cross-domain meter existence validation is a transport-boundary concern, preserving clean domain separation.

<!-- archie:ai-end -->
