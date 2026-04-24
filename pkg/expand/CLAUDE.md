# expand

<!-- archie:ai-start -->

> Generic type-safe expand/field-selection mechanism for API responses. Callers declare an Expandable[T] enum type; Expand[T] is a validated slice of those values with Has/With/Without/SetOrUnsetIf helpers used in service input types to control which nested relations are fetched.

## Patterns

**Expandable enum type** — Define a string-based enum type implementing Expandable[T] with a Values() []T method listing all valid expand keys. Use this as the type parameter for Expand[T] in service input structs. (`type InvoiceExpand string
func (InvoiceExpand) Values() []InvoiceExpand { return []InvoiceExpand{InvoiceExpandLines, InvoiceExpandPayments} }
const InvoiceExpandLines InvoiceExpand = "lines"`)
**Expand.Validate() in input validation** — Call input.Expand.Validate() in service input Validate() methods to reject unknown expand values before passing them to the adapter. (`if err := input.Expand.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Expand.Has() in adapter queries** — Use expand.Has(SomeExpandValue) inside adapter query builders to conditionally eager-load relations via Ent's .WithX() methods. (`q := client.Invoice.Query(); if params.Expand.Has(InvoiceExpandLines) { q = q.WithLines() }`)
**SetOrUnsetIf for conditional expand assembly** — Use SetOrUnsetIf(condition, value) to build expand slices from boolean feature flags without manual append/filter logic. (`expand = expand.SetOrUnsetIf(cfg.LoadPayments, InvoiceExpandPayments)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `expand.go` | Full Expandable interface, Expand[T] type, and all methods. Entire package API. | Expandable requires comparable — string-based enums satisfy this; int-based enums do too, but struct-based expand keys would not compile. |

## Anti-Patterns

- Passing raw string slices for expand fields instead of typed Expand[T] — loses compile-time safety and Validate().
- Skipping Expand.Validate() in service input validation — allows unknown expand values to silently no-op in adapter queries.
- Implementing Values() to return an empty slice — Validate() will reject all expand values including valid ones.

## Decisions

- **Expand is a separate generic package rather than inlined per-domain.** — Multiple domains (billing, subscription, customer) need the same validated expand-slice semantics; a shared package avoids duplicating the Has/With/Without logic.

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
