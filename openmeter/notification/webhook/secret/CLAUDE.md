# secret

<!-- archie:ai-start -->

> Utility package for generating and validating Svix-compatible HMAC signing secrets with the 'whsec_' prefix. No state, no interfaces — pure functions used when creating notification webhook channels.

## Patterns

**whsec_ prefix convention** — Generated secrets are always prefixed with SigningSecretPrefix ('whsec_') followed by base64-encoded random bytes. Validation strips the prefix before checking length and base64 validity. (`return "whsec_" + base64.StdEncoding.EncodeToString(b), nil`)
**Validation via models.NewNillableGenericValidationError** — ValidateSigningSecret collects errors with errors.Join and wraps the result in models.NewNillableGenericValidationError so callers get a nil error (valid) or a typed GenericValidationError (invalid) that maps to HTTP 400. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**Exported size constant for testability** — DefaultSigningSecretSize = 32 is exported so tests and callers can reference the canonical size without magic numbers. (`const DefaultSigningSecretSize = 32`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `secret.go` | All secret generation and validation logic; no other files in this package. | Length bounds in ValidateSigningSecret (32–100 chars after stripping prefix) are Svix API constraints — changing them without checking Svix docs will break webhook channel creation. |

## Anti-Patterns

- Returning a raw error instead of models.NewNillableGenericValidationError — breaks the HTTP 400 mapping chain
- Using encoding/hex or non-standard base64 variants — Svix expects standard base64
- Storing secrets in plain text; this package only generates/validates — persistence is the caller's responsibility

## Decisions

- **Use crypto/rand for secret generation** — Signing secrets are cryptographic material; math/rand would produce predictable values exploitable for HMAC forgery.
- **Wrap validation errors with models.NewNillableGenericValidationError** — Keeps error typing consistent with the rest of the domain so commonhttp.GenericErrorEncoder maps it to HTTP 400 without special-casing this package.

<!-- archie:ai-end -->
