# equal

<!-- archie:ai-start -->

> Defines the Equaler[T] interface and nil-safe pointer comparison helpers, extracted from pkg/models specifically to avoid an import cycle. Backs equality-based diffing and value comparison across domain types.

## Patterns

**Equaler interface duplicated to break a cycle** — Equaler[T any] { Equal(other T) bool } lives here (not only in pkg/models) so low-level packages like pkg/entitydiff can depend on it without importing models. Implement Equal on domain types to satisfy it. (`type Equaler[T any] interface { Equal(other T) bool }`)
**Nil-safe pointer equality helpers** — PtrEqual compares two *T where T is Equaler (both nil equal, one nil unequal); HasherPtrEqual does the same via hasher.Hasher.Hash(). (`equal.PtrEqual(a, b)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `equal.go` | Equaler interface + PtrEqual/HasherPtrEqual | Keep this package dependency-light (only pkg/hasher) — adding heavier imports risks recreating the models import cycle it exists to avoid |

## Anti-Patterns

- Importing pkg/models from here — defeats the cycle-breaking purpose of duplicating Equaler
- Adding domain-specific equality logic; this package holds only the generic interface and pointer helpers

## Decisions

- **Duplicate Equaler from pkg/models instead of sharing** — Documented in code: avoids a circular dependency between models and packages that need the interface

<!-- archie:ai-end -->
