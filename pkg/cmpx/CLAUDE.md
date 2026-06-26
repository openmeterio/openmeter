# cmpx

<!-- archie:ai-start -->

> Tiny generics helper exposing a Comparable[T] interface and a Compare wrapper that delegates to a type's own Compare(T) int method. Used to order domain values deterministically (e.g. ledger collector).

## Patterns

**Self-comparing types** — Types orderable via cmpx must implement Comparable[T] with a Compare(T) int method returning the standard -1/0/1 contract; Compare[T] just forwards to it. (`func Compare[T Comparable[T]](left, right T) int { return left.Compare(right) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `compare.go` | Defines Comparable[T any] interface and the generic Compare function. | Compare assumes left.Compare(right) follows the standard tri-state ordering; a non-conforming Compare implementation silently corrupts sorts. |

## Anti-Patterns

- Adding concrete comparison logic here — this package only abstracts over a type's own Compare method.
- Importing heavy dependencies; keep this dependency-free.

## Decisions

- **Delegate ordering to the value's own Compare method rather than reimplementing comparators.** — Lets domain types own their ordering semantics while generic algorithms remain type-agnostic.

<!-- archie:ai-end -->
