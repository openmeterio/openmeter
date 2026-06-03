# expand

<!-- archie:ai-start -->

> Generic type-safe expand/field-selection mechanism for API responses. Callers declare an Expandable[T] string-enum type; Expand[T] is a validated slice of those values with Has/With/Without/SetOrUnsetIf helpers used in service input types to control which nested Ent relations are eager-loaded.

## Patterns

**Expandable enum type** — Define a string-based type implementing Expandable[T] with Values() []T listing all valid keys; use as the type parameter for Expand[T] in service input structs. (`type InvoiceExpand string
const InvoiceExpandLines InvoiceExpand = "lines"
func (InvoiceExpand) Values() []InvoiceExpand { return []InvoiceExpand{InvoiceExpandLines} }`)
**Expand.Validate() in service input** — Call input.Expand.Validate() inside the input type's Validate() so unknown expand values are rejected before the adapter. (`if err := input.Expand.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Expand.Has() in adapter Ent queries** — Use expand.Has(SomeValue) to conditionally append Ent WithX() eager-load calls. (`q := client.Invoice.Query()
if params.Expand.Has(InvoiceExpandLines) { q = q.WithLines() }`)
**SetOrUnsetIf for conditional assembly** — Build expand slices from boolean flags via SetOrUnsetIf(condition, value) rather than manual append/filter. (`expand = expand.SetOrUnsetIf(cfg.LoadPayments, InvoiceExpandPayments)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `expand.go` | Complete package API: Expandable interface, Expand[T] type, and all methods (Validate, Has, Clone, With, Without, SetOrUnsetIf). | Expandable requires comparable — string enums satisfy this, struct keys do not compile; Values() returning empty slice makes Validate() reject all expand values. |

## Anti-Patterns

- Passing raw string slices for expand fields instead of typed Expand[T] — loses compile-time safety and Validate().
- Skipping Expand.Validate() in input validation — unknown expand values silently no-op in adapter queries.
- Implementing Values() to return an empty slice — Validate() rejects all expand values including valid ones.
- Constructing Expand[T] with non-comparable type parameters — will not compile.

## Decisions

- **Expand is a separate generic package rather than inlined per-domain.** — Billing, subscription, and customer need the same validated expand-slice semantics; a shared package centralizes Has/With/Without and the single-operator validation.

## Example: Define and use a typed expand for invoice queries

```
import "github.com/openmeterio/openmeter/pkg/expand"

type InvoiceExpand string
const InvoiceExpandLines InvoiceExpand = "lines"
func (InvoiceExpand) Values() []InvoiceExpand { return []InvoiceExpand{InvoiceExpandLines} }

type ListInvoicesInput struct { Namespace string; Expand expand.Expand[InvoiceExpand] }
// In adapter:
q := client.Invoice.Query()
if input.Expand.Has(InvoiceExpandLines) { q = q.WithLines() }
```

<!-- archie:ai-end -->
