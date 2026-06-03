# equal

<!-- archie:ai-start -->

> Minimal interface and helpers for structural equality comparison between domain types. Exists as a separate package to break circular dependencies when entitydiff and models both need the Equaler contract.

## Patterns

**Equaler[T] interface** — Implement Equal(other T) bool on value types participating in entitydiff reconciliation or needing pointer-safe equality; don't add it to types that already have a goderive-generated Equal. (`func (l InvoiceLine) Equal(other InvoiceLine) bool { return l.ID == other.ID && l.Amount.Equal(other.Amount) }`)
**PtrEqual for nullable Equaler fields** — Use PtrEqual[T] to compare two *T where both nil means equal, avoiding manual nil guards. (`equal.PtrEqual(a.TaxCode, b.TaxCode)`)
**HasherPtrEqual for hash-comparable types** — Use HasherPtrEqual[T] for types implementing hasher.Hasher to compare by hash digest rather than field-by-field. (`equal.HasherPtrEqual(a.Metadata, b.Metadata)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `equal.go` | Defines Equaler[T] and PtrEqual/HasherPtrEqual; the entire package API. | HasherPtrEqual requires the type to implement pkg/hasher.Hasher — that dependency, not just equal.Equaler. |

## Anti-Patterns

- Duplicating the Equaler interface in another package instead of importing pkg/equal — breaks the entitydiff type constraint.
- Implementing Equal with reflect.DeepEqual — unsafe for pointer fields and billing amount types.

## Decisions

- **Equaler[T] lives in pkg/equal, not pkg/models** — entitydiff imports equal; models imports entitydiff in some domains; a separate shared equal package cuts the import cycle.

<!-- archie:ai-end -->
