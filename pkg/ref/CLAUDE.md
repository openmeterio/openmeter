# ref

<!-- archie:ai-start -->

> Single-file utility providing IDOrKey — a discriminated union type that holds either a ULID-format ID or an arbitrary string key, used wherever domain lookups must accept both identifiers interchangeably.

## Patterns

**ULID-based discrimination** — ParseIDOrKey uses ulid.Parse to decide which field to populate; a successful parse → ID, a parse error → Key. Never set both fields manually. (`ref := ref.ParseIDOrKey(rawParam) // ID set if ULID, Key set otherwise`)
**Validate before use** — Always call Validate() before passing an IDOrKey into a service layer; it returns an error if both fields are empty. (`if err := r.Validate(); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ref.go` | Defines IDOrKey struct with GetIDs, GetKeys, Validate, and ParseIDOrKey helpers. | Callers must not set both ID and Key simultaneously; Validate only guards the all-empty case, not the both-set case. |

## Anti-Patterns

- Setting both ID and Key on an IDOrKey struct — semantics are undefined
- Skipping Validate() before service calls — silently passes empty identifiers
- Adding new identifier types here instead of defining a new discriminated union in the relevant domain package

## Decisions

- **Use ulid.Parse as the discriminator rather than a regex or length check** — ULID has a canonical 26-char Crockford Base32 format; parse error is the cleanest way to distinguish it from freeform key strings without false positives.

<!-- archie:ai-end -->
