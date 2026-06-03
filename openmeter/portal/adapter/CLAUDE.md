# adapter

<!-- archie:ai-start -->

> JWT-backed stateless implementation of portal.Service that issues and validates short-lived HS256 tokens scoped to namespace, subject, and optional meter slug allowlist. ListTokens and InvalidateToken are intentionally unimplemented stubs returning GenericNotImplementedError — no database backing by design.

## Patterns

**Config-validated constructor** — New(Config) validates all fields via Config.Validate() before constructing; always returns (portal.Service, error) — never panics on bad config. (`func New(config Config) (portal.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Interface compliance assertion** — var _ portal.Service = (*adapter)(nil) at package level — compile-time proof the struct satisfies the interface. (`var _ portal.Service = (*adapter)(nil)`)
**JWT claims struct embeds jwt.RegisteredClaims** — JTWPortalTokenClaims embeds jwt.RegisteredClaims and adds Namespace, Id, AllowedMeterSlugs. Always parse with ParseWithClaims into this type, never into map[string]interface{}. (`type JTWPortalTokenClaims struct { jwt.RegisteredClaims; Namespace string; AllowedMeterSlugs []string }`)
**Strict JWT parse options — never relax** — Validate uses WithStrictDecoding, WithExpirationRequired, WithIssuer, and WithValidMethods. Do not remove any when extending validation. (`opts := []jwt.ParserOption{jwt.WithStrictDecoding(), jwt.WithExpirationRequired(), jwt.WithIssuer(PortalTokenIssuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name})}`)
**Noop adapter for disabled/test contexts** — NewNoop() returns a full no-op portal.Service where every method returns models.NewGenericNotImplementedError — not nil error. Use in Wire when portal is disabled. (`func (a *noopAdapter) CreateToken(ctx, input) (*portal.PortalToken, error) { return nil, models.NewGenericNotImplementedError(fmt.Errorf("noop adapter")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, Validate(), New() constructor, adapter struct definition. | Secret is []byte internally; Config.Secret is string — converted in New(). Expire=0 is rejected by Validate(). Always call New() and check the error. |
| `token.go` | CreateToken (JWT sign) and Validate (JWT parse + claims extraction); ListTokens and InvalidateToken stub out with NotImplemented. | The raw token string is only exposed in the CreateToken response (Token field NOT populated in toAPIPortalToken) — adding it elsewhere is a security risk. |
| `noop.go` | Full no-op portal.Service for tests and disabled-portal wiring. | All methods must return models.NewGenericNotImplementedError, not nil — callers distinguish 'not implemented' from success. No business logic here. |

## Anti-Patterns

- Storing token state in a database here — this adapter is intentionally stateless.
- Relaxing JWT validation options (removing WithExpirationRequired, WithStrictDecoding, WithValidMethods, or WithIssuer).
- Populating the Token field inside toAPIPortalToken — token string must only appear in CreateToken response.
- Calling New() without checking the returned error.
- Adding business logic to the noop adapter — it must remain a pure stub returning NotImplementedError.

## Decisions

- **Stateless JWT tokens with no DB persistence for ListTokens/InvalidateToken.** — Short-lived portal metering tokens do not require revocation in the current design; unimplemented methods return NotImplemented to make gaps explicit.
- **HS256 with a shared secret rather than asymmetric signing.** — Portal tokens are validated by the same server that issues them, so symmetric signing suffices and avoids key distribution complexity.

## Example: Create and validate a portal token

```
import (
  "github.com/openmeterio/openmeter/openmeter/portal/adapter"
  "github.com/openmeterio/openmeter/openmeter/portal"
)

svc, err := adapter.New(adapter.Config{Secret: "s3cr3t", Expire: 24 * time.Hour})
if err != nil { ... }
token, _ := svc.CreateToken(ctx, portal.CreateTokenInput{Namespace: "ns", Subject: "user-1"})
// token.Token contains the raw JWT string
claims, _ := svc.Validate(ctx, *token.Token)
```

<!-- archie:ai-end -->
