# defaultx

<!-- archie:ai-start -->

> Tiny generic utility package providing nil-safe pointer dereferencing (WithDefault) and zero-value fallback (IfZero). Used across domain packages to reduce nil-guard boilerplate when handling optional/pointer fields.

## Patterns

**WithDefault for pointer optionals** — Use WithDefault[T](ptr, fallback) whenever a *T may be nil and a non-pointer fallback is needed; never dereference pointers without this guard. (`timeout := defaultx.WithDefault(cfg.Timeout, 30*time.Second)`)
**IfZero for zero-value replacement** — Use IfZero[T](val, fallback) for comparable types where the zero value signals 'unset'. Not for pointers — use WithDefault. (`name := defaultx.IfZero(input.Name, "default")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `default.go` | Defines both exported functions; the entire package API. | IfZero uses the comparable constraint — cannot be used with slices, maps, or non-comparable structs. |

## Anti-Patterns

- Adding non-default-related utilities here — keep the package focused on nil/zero fallbacks.
- Using IfZero with pointer types (zero value of a pointer is nil) — use WithDefault instead.

## Decisions

- **Separate WithDefault (pointer) and IfZero (zero-value comparable)** — Pointer nil-check and zero-value check are distinct semantics; conflating them would require type assertions or reflection.

<!-- archie:ai-end -->
