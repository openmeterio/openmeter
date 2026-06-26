# secret

<!-- archie:ai-start -->

> Generates and validates Svix-style webhook signing secrets used by the notification webhook layer. Pure utility leaf package with no dependencies beyond crypto/rand, encoding/base64, and pkg/models for validation errors.

## Patterns

**whsec_ prefixed base64 secrets** — NewSigningSecret reads `size` random bytes via crypto/rand and returns them base64-encoded with the SigningSecretPrefix ("whsec_"). DefaultSigningSecretSize is 32; use NewSigningSecretWithDefaultSize() for the standard size. (`return "whsec_" + base64.StdEncoding.EncodeToString(b), nil`)
**Aggregate validation via NewNillableGenericValidationError** — ValidateSigningSecret collects all issues into `var errs []error`, then returns models.NewNillableGenericValidationError(errors.Join(errs...)) so callers get nil when valid and a combined error otherwise. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**Strip optional prefix before validating** — Validation CutPrefixes SigningSecretPrefix first, then checks length (32-100 chars) and base64 decodability on the remainder, so secrets validate with or without the prefix. (`s, _ := strings.CutPrefix(secret, SigningSecretPrefix)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `secret.go` | Sole file: DefaultSigningSecretSize/SigningSecretPrefix constants, NewSigningSecret(size), NewSigningSecretWithDefaultSize(), and ValidateSigningSecret(secret). | Length bounds (32-100) are checked on the prefix-stripped string; keep crypto/rand (not math/rand); base64.StdEncoding must match between generation and decode validation. |

## Anti-Patterns

- Using math/rand instead of crypto/rand for secret generation.
- Returning on the first validation failure instead of joining all errs.
- Hardcoding the "whsec_" literal in callers instead of using SigningSecretPrefix.
- Mixing base64 encodings (URL vs Std) between NewSigningSecret and ValidateSigningSecret.

## Decisions

- **Mirror Svix's whsec_-prefixed base64 secret format.** — The notification webhook layer integrates with Svix, so signing secrets must be interoperable with Svix's expected format.

## Example: Validate an incoming signing secret with or without the prefix

```
func ValidateSigningSecret(secret string) error {
	var errs []error
	s, _ := strings.CutPrefix(secret, SigningSecretPrefix)
	if len(s) < 32 || len(s) > 100 {
		errs = append(errs, errors.New("secret length must be between 32 to 100 chars without the optional prefix"))
	}
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		errs = append(errs, errors.New("invalid base64 string"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

<!-- archie:ai-end -->
