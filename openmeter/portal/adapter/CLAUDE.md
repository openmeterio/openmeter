# adapter

<!-- archie:ai-start -->

> Adapter implementation of the portal.Service interface for issuing and validating namespace-scoped JWT portal tokens (HS256). Tokens grant subjects read access to specific meter slugs; this is the only place tokens are minted/verified.

## Patterns

**Config.Validate + New constructor** — New(config Config) validates before constructing; non-empty Secret and non-zero Expire are mandatory. (`func New(config Config) (portal.Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{secret: []byte(config.Secret), expire: config.Expire}, nil }`)
**Compile-time interface assertion** — Every adapter declares `var _ portal.Service = (*adapter)(nil)` so it must implement the full interface. (`var _ portal.Service = (*adapter)(nil)`)
**JWT minted with fixed issuer + HS256** — CreateToken signs JTWPortalTokenClaims with jwt.SigningMethodHS256, Issuer=PortalTokenIssuer ("openmeter"), embedding Namespace, Id (uuid), Subject, AllowedMeterSlugs. (`jwt.NewWithClaims(jwt.SigningMethodHS256, JTWPortalTokenClaims{Namespace: input.Namespace, Id: id, RegisteredClaims: jwt.RegisteredClaims{Issuer: PortalTokenIssuer, ...}})`)
**Strict parser options on Validate** — Validate parses with WithStrictDecoding, WithExpirationRequired, WithIssuer(PortalTokenIssuer), WithValidMethods([HS256]); never relax these. (`opts := []jwt.ParserOption{jwt.WithStrictDecoding(), jwt.WithExpirationRequired(), jwt.WithIssuer(PortalTokenIssuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name})}`)
**Input validation delegated to input type** — Handler methods call input.Validate() (e.g. CreateToken calls input.Validate()) rather than re-implementing field checks. (`if err := input.Validate(); err != nil { return nil, err }`)
**Unimplemented ops return NewGenericNotImplementedError** — ListTokens and InvalidateToken (stateless JWT, no store) return models.NewGenericNotImplementedError; noopAdapter returns it for everything. (`return resp, models.NewGenericNotImplementedError(fmt.Errorf("listing tokens"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct (Secret, Expire), Validate(), New() factory, and the adapter struct holding secret []byte + expire time.Duration. | Secret is stored as []byte; both fields are required by Validate — adding a code path that bypasses New skips validation. |
| `token.go` | JTWPortalTokenClaims type plus CreateToken/Validate/ListTokens/InvalidateToken adapter methods. | ListTokens/InvalidateToken are intentionally not-implemented because tokens are stateless JWTs with no backing store; do not stub fake success. ExpiresAt defaults to now+expire unless input overrides. |
| `noop.go` | noopAdapter via NewNoop() returning portal.Service whose every method errors with NotImplemented — used when portal is disabled. | Keep noop method set in sync with the portal.Service interface or the var-assertion would fail elsewhere. |

## Anti-Patterns

- Relaxing JWT parser options (dropping WithIssuer/WithValidMethods/WithExpirationRequired) — opens token forgery / alg confusion.
- Mapping the raw token string in any output except CreateToken — the token secret must only leave the system once at creation.
- Constructing adapter directly instead of via New(), bypassing Config.Validate.
- Returning fake success from ListTokens/InvalidateToken; they are unimplemented by design (stateless JWT).

## Decisions

- **Portal tokens are stateless signed JWTs, not DB rows.** — No revocation store exists, so ListTokens/InvalidateToken return NotImplemented rather than pretending to manage persisted state.
- **A noop adapter shares the package.** — Lets DI wire a portal.Service that hard-fails when the portal feature is disabled, keeping the interface non-nil.

## Example: Mint a namespace-scoped portal token

```
token := jwt.NewWithClaims(jwt.SigningMethodHS256, JTWPortalTokenClaims{
  Namespace: input.Namespace,
  Id:        uuid.New().String(),
  RegisteredClaims: jwt.RegisteredClaims{
    Subject:   input.Subject,
    ExpiresAt: jwt.NewNumericDate(expiresAt),
    IssuedAt:  jwt.NewNumericDate(time.Now()),
    Issuer:    PortalTokenIssuer,
  },
  AllowedMeterSlugs: *allowedMeterSlugs,
})
tokenString, err := token.SignedString(a.secret)
```

<!-- archie:ai-end -->
