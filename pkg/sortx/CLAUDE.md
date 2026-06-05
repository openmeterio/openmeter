# sortx

<!-- archie:ai-start -->

> Defines the project-wide sort Order string enum (ASC/DESC) shared by list/pagination APIs and adapters across billing, customer, notification, productcatalog, and api/v3.

## Patterns

**Order string enum with default** — Order is a string type with OrderAsc="ASC", OrderDesc="DESC", OrderDefault=OrderAsc, OrderNone="". Use these constants, never raw strings. (`order := sortx.OrderDesc`)
**IsDefaultValue means unset** — IsDefaultValue() returns true only when Order==OrderNone (empty), used to decide whether to apply a fallback ordering. (`if order.IsDefaultValue() { order = sortx.OrderDefault }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `order.go` | Entire package: Order type, the four constants, String(), IsDefaultValue(). | A TODO notes IsDefaultValue is misnamed — it checks for unset (OrderNone), not for OrderDefault. Don't assume it returns true for OrderAsc. |

## Anti-Patterns

- Comparing against literal "ASC"/"DESC" strings instead of the constants.
- Treating IsDefaultValue() as 'equals OrderDefault' — it only tests OrderNone.

## Decisions

- **Single shared enum rather than per-package order types.** — 229+ importers across api/v3 and openmeter domains reuse one canonical Order so list endpoints stay consistent.

## Example: Default an unset order before querying

```
import "github.com/openmeterio/openmeter/pkg/sortx"

if order.IsDefaultValue() {
    order = sortx.OrderDefault
}
```

<!-- archie:ai-end -->
