# equal

<!-- archie:ai-start -->

> Minimal interface and helpers for structural equality comparison between domain types. Exists as a separate package to break circular dependencies when entitydiff and models both need the Equaler contract.

## Patterns

**Equaler[T] interface** — Implement Equal(other T) bool on value types that participate in entitydiff reconciliation or need pointer-safe equality. Do not implement on types that already embed a generated Equal from goderive. (`func (l InvoiceLine) Equal(other InvoiceLine) bool { return l.ID == other.ID && l.Amount.Equal(other.Amount) }`)
**PtrEqual for nullable Equaler fields** — Use PtrEqual[T] when comparing two *T fields where both nil means equal. Avoids manual nil guard at call sites. (`equal.PtrEqual(a.TaxCode, b.TaxCode)`)
**HasherPtrEqual for hash-comparable types** — Use HasherPtrEqual[T] for types that implement hasher.Hasher instead of Equaler to compare by hash digest rather than field-by-field. (`equal.HasherPtrEqual(a.Metadata, b.Metadata)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `equal.go` | Defines Equaler[T] interface and PtrEqual/HasherPtrEqual helpers. Entire package API. | HasherPtrEqual requires the type to implement pkg/hasher.Hasher — adding a new type here requires that dependency, not just equal.Equaler. |

## Anti-Patterns

- Duplicating the Equaler interface in another package instead of importing pkg/equal — this breaks the entitydiff type constraint
- Implementing Equal with reflect.DeepEqual — breaks with pointer fields and is not safe for billing amount types

## Decisions

- **Equaler[T] lives in pkg/equal rather than pkg/models to avoid import cycles between entitydiff and models** — entitydiff imports equal; models imports entitydiff in some domains; a single shared equal package cuts the cycle.

<!-- archie:ai-end -->
