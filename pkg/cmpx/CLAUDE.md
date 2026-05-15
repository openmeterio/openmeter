# cmpx

<!-- archie:ai-start -->

> Single-file package exposing a Comparable[T] constraint and a Compare[T] generic function for deterministic ordering of domain types. Used wherever two values need to be ordered without boilerplate type assertions.

## Patterns

**Implement Comparable[T] on domain types** — Any type that needs ordering must implement Compare(T) int. The generic Compare[T] function dispatches to left.Compare(right). (`func (a MyType) Compare(b MyType) int { return cmpx.Compare(a, b) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `compare.go` | Defines Comparable[T] interface and Compare[T] generic helper. No dependencies other than stdlib. | The package is intentionally minimal — do not add non-ordering utilities here. |

## Anti-Patterns

- Adding any logic beyond comparison ordering to this package
- Using reflect.DeepEqual or sort.Slice comparisons in callers instead of implementing Comparable[T]

<!-- archie:ai-end -->
