# expand

<!-- archie:ai-start -->

> Generic type-safe expand/field-selection mechanism for API responses. Callers declare an Expandable[T] string-enum type; Expand[T] is a validated slice of those values with Has/With/Without/SetOrUnsetIf helpers used in service input types to control which nested Ent relations are eager-loaded.

## Patterns

**Expandable enum type** — Define a string-based type implementing Expandable[T] with a Values() []T method listing all valid keys. Use as the type parameter for Expand[T] in service input structs. (`type InvoiceExpand string
const InvoiceExpandLines InvoiceExpand = "lines"
func (InvoiceExpand) Values() []InvoiceExpand { return []InvoiceExpand{InvoiceExpandLines} }`)
**Expand.Validate() in service input** — Call input.Expand.Validate() inside the input type's Validate() method so unknown expand values are rejected before reaching the adapter. (`if err := input.Expand.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Expand.Has() in adapter Ent queries** — Use expand.Has(SomeExpandValue) to conditionally append Ent WithX() eager-load calls on the query builder. (`q := client.Invoice.Query()
if params.Expand.Has(InvoiceExpandLines) { q = q.WithLines() }`)
**SetOrUnsetIf for conditional assembly** — Build expand slices from boolean feature flags via SetOrUnsetIf(condition, value) rather than manual append/filter logic. (`expand = expand.SetOrUnsetIf(cfg.LoadPayments, InvoiceExpandPayments)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `expand.go` | Complete package API: Expandable interface, Expand[T] type, and all methods (Validate, Has, Clone, With, Without, SetOrUnsetIf). | Expandable requires comparable — string-based enums satisfy this; struct-based expand keys do not compile. Values() returning an empty slice causes Validate() to reject all expand values. |

## Anti-Patterns

- Passing raw string slices for expand fields instead of typed Expand[T] — loses compile-time safety and Validate().
- Skipping Expand.Validate() in service input validation — allows unknown expand values to silently no-op in adapter queries.
- Implementing Values() to return an empty slice — Validate() will reject all expand values including valid ones.
- Constructing Expand[T] values with non-comparable type parameters — will not compile.

## Decisions

- **Expand is a separate generic package rather than inlined per-domain.** — Multiple domains (billing, subscription, customer) need the same validated expand-slice semantics; a shared package avoids duplicating Has/With/Without logic and centralises the single-operator validation.

## Example: Define and use a typed expand for invoice queries

```
import "github.com/openmeterio/openmeter/pkg/expand"

type InvoiceExpand string
const InvoiceExpandLines InvoiceExpand = "lines"
func (InvoiceExpand) Values() []InvoiceExpand { return []InvoiceExpand{InvoiceExpandLines} }

type ListInvoicesInput struct {
    Namespace string
    Expand    expand.Expand[InvoiceExpand]
}

// In adapter:
q := client.Invoice.Query()
if input.Expand.Has(InvoiceExpandLines) {
    q = q.WithLines()
// ...
```

<!-- archie:ai-end -->
