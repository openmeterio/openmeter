# sortx

<!-- archie:ai-start -->

> Minimal single-file package defining the sortx.Order string enum (ASC/DESC) and default/none sentinels used as a shared sort-direction type across all domain list and query inputs.

## Patterns

**Use OrderDefault for unspecified sort direction** — When a caller does not supply a sort direction, default to sortx.OrderDefault (= OrderAsc). Never use the empty string literal directly. (`order := sortx.OrderDefault`)
**IsDefaultValue detects absence, not ascending direction** — Order.IsDefaultValue() returns true only when Order == OrderNone (empty string), not when Order == OrderAsc. Use it solely to detect 'no order specified'. (`if order.IsDefaultValue() { order = sortx.OrderDefault }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `order.go` | Defines Order type and four constants: OrderAsc ('ASC'), OrderDesc ('DESC'), OrderDefault (= OrderAsc), OrderNone (empty string). | IsDefaultValue() checks for OrderNone, not OrderAsc — the name is acknowledged as misleading in a TODO comment. Do not use it to check 'is ascending'. |

## Anti-Patterns

- Adding sort-field definitions or multi-column sort structs here — this package is intentionally a single primitive
- Comparing Order to string literals 'ASC'/'DESC' directly — use the exported constants
- Using IsDefaultValue() as a semantic check for 'is ascending' — it only detects the unset (empty) state

## Decisions

- **Single file, single type, no sub-packages** — Sort direction is a leaf primitive shared by many list inputs; keeping it minimal prevents coupling and circular imports.

<!-- archie:ai-end -->
