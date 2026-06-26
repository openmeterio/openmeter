# defaultx

<!-- archie:ai-start -->

> Tiny generic helpers for defaulting optional values: WithDefault dereferences a pointer or returns a fallback, IfZero substitutes a fallback for the zero value. Used mainly in httpdriver/service layers to normalize optional API inputs.

## Patterns

**Pointer-or-default for optional fields** — WithDefault[T any](value *T, def T) T returns *value when non-nil, else def. Use for optional API request pointers. (`limit := defaultx.WithDefault(req.Limit, 100)`)
**Zero-or-default for comparable values** — IfZero[T comparable](val, def T) T returns def when val equals the zero value. Use for non-pointer optionals like empty strings. (`currency := defaultx.IfZero(in.Currency, "USD")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `default.go` | Both generic helpers (WithDefault, IfZero) | IfZero requires T comparable; cannot be used on slices/maps/funcs |

## Anti-Patterns

- Adding non-trivial logic here — this package is intentionally two stateless generic functions

<!-- archie:ai-end -->
