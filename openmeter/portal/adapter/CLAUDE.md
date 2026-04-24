# adapter

<!-- archie:ai-start -->

> JWT-backed implementation of portal.Service that issues and validates short-lived HS256 tokens scoped to a namespace, subject, and optional meter slug allowlist. No database — tokens are stateless JWTs; ListTokens and InvalidateToken are intentionally unimplemented (return GenericNotImplementedError).

## Patterns

**Config-validated constructor** — New(Config) validates all fields via Config.Validate() before constructing the adapter; return (portal.Service, error) — never panic on bad config. (`func New(config Config) (portal.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Interface compliance assertion** — var _ portal.Service = (*adapter)(nil) at package level — compile-time proof the struct satisfies the interface. (`var _ portal.Service = (*adapter)(nil)`)
**Noop for disabled/unimplemented methods** — Methods not yet backed by persistent storage return models.NewGenericNotImplementedError; noop.go provides a full no-op implementation for test/disabled contexts via NewNoop(). (`func (a *adapter) ListTokens(...) (pagination.Result[*portal.PortalToken], error) { return ..., models.NewGenericNotImplementedError(fmt.Errorf("listing tokens")) }`)
**JWT claims struct embeds jwt.RegisteredClaims** — JTWPortalTokenClaims embeds jwt.RegisteredClaims and adds domain fields (Namespace, Id, AllowedMeterSlugs). Parse with ParseWithClaims into this type, not into map[string]interface{}. (`type JTWPortalTokenClaims struct { jwt.RegisteredClaims; Namespace string; AllowedMeterSlugs []string }`)
**Strict JWT parse options** — Validate uses WithStrictDecoding, WithExpirationRequired, WithIssuer, and WithValidMethods — never relax these constraints when extending validation. (`opts := []jwt.ParserOption{jwt.WithStrictDecoding(), jwt.WithExpirationRequired(), jwt.WithIssuer(PortalTokenIssuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name})}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, Validate(), New() constructor, adapter struct definition | Secret must be []byte internally; Config.Secret is string — conversion happens in New(). Expire=0 is rejected. |
| `token.go` | CreateToken (JWT sign) and Validate (JWT parse + claims extraction); ListTokens and InvalidateToken stub out with NotImplemented | Token field is intentionally NOT populated in toAPIPortalToken mapping — the raw token string is only exposed in CreateToken response. Adding it elsewhere is a security risk. |
| `noop.go` | Full no-op portal.Service for tests and disabled-portal wiring | All methods must return models.NewGenericNotImplementedError, not nil error — callers distinguish 'not implemented' from success. |

## Anti-Patterns

- Storing token state in a database here — this adapter is intentionally stateless; persistence belongs in a future DB-backed adapter
- Relaxing JWT validation options (removing WithExpirationRequired, WithStrictDecoding, etc.)
- Populating the Token field inside toAPIPortalToken — token string must only appear in CreateToken response
- Calling New() without checking the returned error
- Adding business logic to the noop adapter — it must remain a pure stub

## Decisions

- **Stateless JWT tokens with no DB persistence for ListTokens/InvalidateToken** — Short-lived portal tokens for metering access do not require revocation in the current design; unimplemented methods return NotImplemented to make gaps explicit rather than silently succeeding.
- **HS256 with a shared secret rather than asymmetric signing** — Portal tokens are validated by the same server that issues them, so symmetric signing is sufficient and avoids key distribution complexity.

## Example: Creating and validating a portal token

```
import (
	"github.com/openmeterio/openmeter/openmeter/portal/adapter"
	"github.com/openmeterio/openmeter/openmeter/portal"
)

svc, err := adapter.New(adapter.Config{Secret: "s3cr3t", Expire: 24 * time.Hour})
if err != nil { ... }

token, err := svc.CreateToken(ctx, portal.CreateTokenInput{
	Namespace: "ns",
	Subject:   "user-1",
})
// token.Token contains the raw JWT string

claims, err := svc.Validate(ctx, *token.Token)
```

<!-- archie:ai-end -->
