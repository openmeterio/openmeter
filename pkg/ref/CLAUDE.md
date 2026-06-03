# ref

<!-- archie:ai-start -->

> Single-file utility providing IDOrKey — a discriminated union type holding either a ULID-format ID or an arbitrary string key, used wherever domain lookups must accept both identifiers interchangeably without ambiguity.

## Patterns

**ULID-based discrimination via ParseIDOrKey** — ParseIDOrKey uses ulid.Parse to decide which field to populate: successful parse → ID set, parse error → Key set. Never set both fields manually. (`r := ref.ParseIDOrKey(rawParam) // r.ID set if ULID, r.Key set otherwise`)
**Validate before service call** — Always call Validate() before passing an IDOrKey into a service layer. It errors if both fields are empty (the all-zero case). (`if err := r.Validate(); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ref.go` | Defines IDOrKey with GetIDs, GetKeys, Validate, and ParseIDOrKey — the entire package surface. | Validate only guards the all-empty case, not the both-set case. Callers must not set both ID and Key — semantics are undefined and unchecked. |

## Anti-Patterns

- Setting both ID and Key on an IDOrKey struct — undefined semantics, no runtime guard.
- Skipping Validate() before service calls — silently passes empty identifiers into domain logic.
- Adding new identifier union types here — define domain-specific discriminated unions in the relevant domain package instead.

## Decisions

- **Use ulid.Parse as the discriminator rather than a regex or length check.** — ULID has a canonical 26-char Crockford Base32 format; a parse error cleanly distinguishes it from freeform key strings without false positives from length heuristics.

<!-- archie:ai-end -->
