# secret

<!-- archie:ai-start -->

> Pure-function utility package for generating and validating Svix-compatible HMAC signing secrets with the 'whsec_' prefix. No state, no interfaces — used exclusively when creating notification webhook channels.

## Patterns

**whsec_ prefix convention** — Generated secrets are always prefixed with SigningSecretPrefix ('whsec_') followed by base64-encoded random bytes. ValidateSigningSecret strips the prefix before checking length (32–100 chars) and base64 validity — these bounds are Svix API constraints. (`return "whsec_" + base64.StdEncoding.EncodeToString(b), nil`)
**Validation returns models.NewNillableGenericValidationError** — ValidateSigningSecret collects errors with errors.Join and wraps with models.NewNillableGenericValidationError, returning nil when valid and a typed GenericValidationError (HTTP 400) when invalid. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**Exported size constant for testability** — DefaultSigningSecretSize = 32 is exported so tests and callers reference the canonical size without magic numbers. (`const DefaultSigningSecretSize = 32`)
**crypto/rand for secret generation** — NewSigningSecret uses crypto/rand.Read — signing secrets are cryptographic HMAC material and must be unpredictable. (`_, err := rand.Read(b)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `secret.go` | All secret generation and validation logic; single file, no other files in this package. | Length bounds in ValidateSigningSecret (32–100 chars after stripping prefix) are Svix API constraints — changing them without verifying against Svix docs will break webhook channel creation. |

## Anti-Patterns

- Returning a raw error instead of models.NewNillableGenericValidationError — breaks HTTP 400 mapping in GenericErrorEncoder
- Using encoding/hex or non-standard base64 variants — Svix expects standard base64 (StdEncoding)
- Using math/rand instead of crypto/rand — produces predictable secrets exploitable for HMAC forgery
- Storing secrets in this package — it only generates/validates; persistence is the caller's responsibility

## Decisions

- **Wrap validation errors with models.NewNillableGenericValidationError** — Keeps error typing consistent with the rest of the notification domain so commonhttp.GenericErrorEncoder maps validation failures to HTTP 400 without special-casing this package.
- **Prefix all secrets with 'whsec_'** — Svix interprets this prefix to select HMAC-SHA256 signing; secrets without the prefix use a different (weaker) algorithm.

## Example: Generate and validate a signing secret for a new webhook channel

```
import (
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
)

sec, err := secret.NewSigningSecretWithDefaultSize()
if err != nil {
	return fmt.Errorf("generate signing secret: %w", err)
}
if err := secret.ValidateSigningSecret(sec); err != nil {
	return err // GenericValidationError -> HTTP 400
}
```

<!-- archie:ai-end -->
