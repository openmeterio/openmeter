# cmpx

<!-- archie:ai-start -->

> Single-file package exposing a Comparable[T] constraint and a Compare[T] generic function for deterministic ordering of domain types, without boilerplate type assertions.

## Patterns

**Implement Comparable[T] on domain types** — Any type needing ordering implements Compare(T) int; the generic Compare[T] function dispatches to left.Compare(right). (`func Compare[T Comparable[T]](left, right T) int { return left.Compare(right) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `compare.go` | Defines Comparable[T] interface and Compare[T] generic helper. No dependencies beyond stdlib. | Intentionally minimal — do not add non-ordering utilities here. |

## Anti-Patterns

- Adding any logic beyond comparison ordering to this package.
- Using reflect.DeepEqual or sort.Slice comparisons in callers instead of implementing Comparable[T].

<!-- archie:ai-end -->
