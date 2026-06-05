# portal

<!-- archie:ai-start -->

> Domain root for the customer-facing portal: defines portal.Service for minting/validating namespace-scoped, meter-slug-restricted JWT tokens that grant subjects read access. Children split into adapter (HS256 token mint/verify), authenticator (OpenAPI-driven Bearer middleware), and httphandler (token CRUD transport).

## Patterns

**Service is a token-management interface** — service.go declares Service = PortalTokenService{CreateToken, Validate, ListTokens, InvalidateToken}. Tokens are stateless signed JWTs, so ListTokens/InvalidateToken are unimplemented by design in the adapter. (`type PortalTokenService interface { CreateToken(...); Validate(...); ListTokens(...); InvalidateToken(...) }`)
**Inputs validate by accumulation** — CreateTokenInput/ListTokensInput/InvalidateTokenInput each implement Validate() collecting into var errs []error and returning errors.Join(errs...); empty errs returns nil. (`if i.Namespace == "" { errs = append(errs, fmt.Errorf("namespace is required")) }`)
**Token string only ever leaves on creation** — PortalToken.Token is a *string documented "Only set when creating a token"; the standard mapper omits it so list/get never leak the secret. (`Token *string // Only set when creating a token.`)
**Empty AllowedMeterSlugs means all meters** — PortalTokenClaims.AllowedMeterSlugs empty list = all slugs allowed; CreateTokenInput.Validate rejects individual empty-string slugs but accepts an empty list. (`for _, slug := range *i.AllowedMeterSlugs { if slug == "" { errs = append(errs, fmt.Errorf("allowed meter slug cannot be empty")) } }`)
**InvalidateToken needs id or subject** — InvalidateTokenInput.Validate requires at least one of ID/Subject and rejects empty-but-present values. (`if i.ID == nil && i.Subject == nil { errs = append(errs, fmt.Errorf("either id or subject must be provided")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service/PortalTokenService interfaces, PortalToken/PortalTokenClaims types, and the three Input structs with Validate(). | ListTokens/InvalidateToken are stateless-JWT no-ops in the adapter — do not assume DB-backed behavior. PortalToken.Token must stay omitted from list/get mappings. |

## Anti-Patterns

- Mapping PortalToken.Token into any response other than CreateToken — leaks the secret.
- Relaxing JWT parser options (WithIssuer/WithValidMethods/WithExpirationRequired) in the adapter — enables forgery / alg confusion.
- Returning fake success from ListTokens/InvalidateToken instead of the by-design not-implemented behavior.
- Creating a token without validating AllowedMeterSlugs against meter.Service in the handler.
- Reading the namespace from request params instead of the namespace decoder.

## Decisions

- **Portal tokens are stateless HS256 JWTs, not DB rows.** — Tokens are self-validating (namespace + subject + allowed meter slugs + expiry in claims), so listing/invalidation of arbitrary tokens is not supported.
- **Authentication is generic over OpenAPI 3 security requirements rather than hardcoded per route.** — The authenticator resolves schemes by name from api.PortalTokenAuthScopes and tries requirements in order, so adding/removing auth is spec-driven.

## Example: CreateTokenInput validation with joined errors

```
func (i CreateTokenInput) Validate() error {
	var errs []error
	if i.Namespace == "" { errs = append(errs, fmt.Errorf("namespace is required")) }
	if i.Subject == "" { errs = append(errs, fmt.Errorf("subject is required")) }
	if i.ExpiresAt != nil && i.ExpiresAt.Before(time.Now()) {
		errs = append(errs, fmt.Errorf("expiration date must be in the future"))
	}
	if len(errs) > 0 { return errors.Join(errs...) }
	return nil
}
```

<!-- archie:ai-end -->
