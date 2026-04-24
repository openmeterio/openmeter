# sortx

<!-- archie:ai-start -->

> Minimal package defining the sortx.Order string enum (ASC/DESC/empty) used as a shared sort-direction type across all domain list/query inputs.

## Patterns

**Use OrderDefault for unspecified sort direction** — When a caller does not supply a sort direction, default to sortx.OrderDefault (= OrderAsc); never use the empty string literal directly. (`order := sortx.OrderDefault`)
**IsDefaultValue checks for unset, not for ASC** — Order.IsDefaultValue() returns true when Order == OrderNone (empty string), not when Order == OrderAsc. The method name is misleading per the in-code TODO; use it only to detect absence of an explicit order. (`if order.IsDefaultValue() { order = sortx.OrderDefault }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `order.go` | Defines Order type and four constants: OrderAsc, OrderDesc, OrderDefault (=OrderAsc), OrderNone (empty). | IsDefaultValue() checks for OrderNone, not OrderDefault — the name is acknowledged as misleading in a TODO comment. |

## Anti-Patterns

- Adding sort-field definitions or multi-column sort structs here — this package is intentionally minimal
- Comparing Order to string literals 'ASC'/'DESC' directly — use the constants
- Using IsDefaultValue() as a semantic check for 'is ascending' — it only detects unset

## Decisions

- **Single file, single type — no sub-packages** — Sort direction is a leaf primitive shared by many list inputs; keeping it minimal prevents coupling and circular imports.

<!-- archie:ai-end -->
