# portal

<!-- archie:ai-start -->

> Issues and validates short-lived HS256 JWT portal tokens scoped to a namespace, subject, and optional meter slug allowlist; ListTokens and InvalidateToken are intentionally unimplemented. The root package owns the Service interface and input/output types.

## Patterns

**Input Validate() before service call** — CreateTokenInput, ListTokensInput, and InvalidateTokenInput each expose Validate() with errors.Join. The adapter and httphandler must call Validate() before delegating. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Token field set only in CreateToken response** — PortalToken.Token is a *string that is nil in all read paths. The JWT string is assigned only after token creation — never inside toAPIPortalToken mapping. (`token, err := adapter.CreateToken(ctx, input); token.Token = &jwtString`)
**Noop adapter for unimplemented methods** — ListTokens and InvalidateToken on the JWT adapter return models.NewGenericNotImplementedError. A separate noop.go provides a full no-op stub for test wiring. (`func (a *Adapter) ListTokens(...) (pagination.Result[*PortalToken], error) { return pagination.Result[*PortalToken]{}, models.NewGenericNotImplementedError("list portal tokens") }`)
**Meter slug allowlist enforced in httphandler operation, not adapter** — CreateToken httphandler validates AllowedMeterSlugs against meter.Service before calling portal.Service.CreateToken. (`// In httphandler/portal.go operation func: validate each slug via meterSvc.GetMeterByIDOrSlug`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service interface (composed of PortalTokenService), all input/output types, and Validate() methods. | AllowedMeterSlugs is *[]string — nil means all slugs allowed, non-nil empty slice means no slugs allowed. Distinguish these cases. |
| `adapter/adapter.go` | JWT-backed stateless adapter. Config.SharedSecret must be non-empty. Strict JWT parse options must not be relaxed. | Do not store token state in DB — this adapter is intentionally stateless. |
| `authenticator/authenticator.go` | Chi middleware dispatching per OpenAPI security scheme; injects validated subject into context. | Auth failures must always be 401, never 200 or 500. |

## Anti-Patterns

- Relaxing JWT validation options (removing WithExpirationRequired, WithStrictDecoding)
- Setting Token field inside toAPIPortalToken — must only be set in CreateToken response path
- Adding DB persistence to the JWT adapter — stateless by design; DB-backed impl would be a new adapter
- Skipping AllowedMeterSlugs validation in httphandler when meterSlug param is absent
- Returning 500 for auth failures in authenticator — must always be 401

## Decisions

- **Stateless JWT tokens with no DB persistence** — Portal tokens are short-lived and scoped; avoiding a DB table removes a write-path dependency and keeps token issuance fast. ListTokens/InvalidateToken are explicitly unimplemented until a DB-backed adapter is added.
- **Meter slug validation in HTTP handler, not portal.Service** — portal.Service has no dependency on meter.Service; meter existence validation is a cross-domain concern handled at the transport boundary.

<!-- archie:ai-end -->
