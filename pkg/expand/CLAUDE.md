# expand

<!-- archie:ai-start -->

> Single-file generic helper for validated, immutable expand-option lists (the `?expand=` query semantics). `Expand[T]` is a slice of self-describing enum-like values that knows its own legal value set via the `Expandable[T]` interface.

## Patterns

**Self-describing expandable type** — T must satisfy `Expandable[T]` = `comparable` + `Values() []T`; validation and enumeration are driven off `empty.Values()`, never a hard-coded list (`type Expand[T Expandable[T]] []T; values := (*new(T)).Values()`)
**Immutable mutators return clones** — `With`, `Without`, `SetOrUnsetIf` never mutate the receiver — they `Clone()` first (copy into a new slice) and return the new value (`func (e Expand[T]) With(v T) Expand[T] { cloned := e.Clone(); ... }`)
**Validate joins per-item errors** — `Validate()` accumulates `var errs []error` and returns `errors.Join(errs...)`; one error per invalid value rather than failing fast (`errs = append(errs, fmt.Errorf("invalid expand value: %v", item)); return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `expand.go` | Whole package: `Expand[T]` slice type plus `Validate`, `Has`, `Clone`, `With`, `Without`, `SetOrUnsetIf` | `With` dedupes (no-op if value present); `Without` uses `lo.Filter` and does NOT clone first — but it returns a new slice so the receiver is still untouched |

## Anti-Patterns

- Mutating an Expand slice in place instead of using the clone-returning mutators
- Hard-coding the valid value list anywhere instead of relying on T.Values()
- Returning on the first invalid item in Validate instead of joining all errors

## Decisions

- **Validation source-of-truth is the type's own Values() method** — Keeps the legal expand set co-located with the enum type, so new expand values can never drift out of sync with the validator

<!-- archie:ai-end -->
