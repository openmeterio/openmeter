# defaultx

<!-- archie:ai-start -->

> Tiny generic utility package providing nil-safe pointer dereferencing (WithDefault) and zero-value fallback (IfZero) helpers. Used across domain packages to reduce nil guard boilerplate when handling optional/pointer fields.

## Patterns

**WithDefault for pointer optionals** — Use WithDefault[T](ptr, fallback) whenever a *T field may be nil and a non-pointer fallback value is needed. Never dereference pointers without this guard. (`timeout := defaultx.WithDefault(cfg.Timeout, 30*time.Second)`)
**IfZero for zero-value replacement** — Use IfZero[T](val, fallback) for comparable types where the zero value signals 'unset'. Do not use for pointer types — use WithDefault instead. (`name := defaultx.IfZero(input.Name, "default")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `default.go` | Defines both exported functions; entire package API lives here. | IfZero uses comparable constraint — cannot be used with slices, maps, or non-comparable structs. |

## Anti-Patterns

- Adding non-default-related utilities here — this package must stay focused on nil/zero fallbacks
- Using IfZero with pointer types (zero value of a pointer is nil, use WithDefault instead)

## Decisions

- **Separate WithDefault (pointer) and IfZero (zero-value comparable) into two functions** — Pointer nil-check and zero-value check are distinct semantics; conflating them into one function would require type assertions or reflection.

<!-- archie:ai-end -->
