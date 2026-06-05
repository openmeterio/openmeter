# filter

<!-- archie:ai-start -->

> The codebase's reusable AIP-style query-filter primitives: typed per-field filter structs (`FilterString`, `FilterInteger`, `FilterFloat`, `FilterTime`, `FilterTimeUnix`, `FilterBoolean`, `FilterULID`) using MongoDB-like `$eq/$ne/$in/$like/$and/$or` JSON tags, each able to validate itself and emit both Ent selector predicates and go-sqlbuilder WHERE expressions. Heavily depended on (~53 in-edges) by v3 handlers and adapters.

## Patterns

**Filter interface contract** — Every filter type satisfies `Filter`: `Validate()`, `ValidateWithComplexity(maxDepth)`, `Select(field) func(*sql.Selector)`, `SelectWhereExpr(field, *sqlbuilder.SelectBuilder) string`, `IsEmpty()`. Compile-time `var _ Filter = (*FilterX)(nil)` assertions enforce this (`var _ Filter = (*FilterString)(nil)`)
**Dual SQL emission, single operator per filter** — `Select` (Ent dialect/sql) and `SelectWhereExpr` (huandu/go-sqlbuilder) must produce equivalent SQL; exactly one operator field may be set or validation returns ErrFilterMultipleOperators (`validateSingleOperator`) (`case f.Eq != nil: return sql.FieldEQ(field, *f.Eq)`)
**Public Validate wraps once via models** — Public `Validate`/`ValidateWithComplexity` call an internal `validateWithComplexity` that returns raw sentinel errors, then wrap exactly once with `models.NewNillableGenericValidationError` (`return models.NewNillableGenericValidationError(f.validateWithComplexity(math.MaxInt))`)
**Recursive And/Or with depth budget** — `$and`/`$or` are `*[]FilterX`; recursion decrements maxDepth and returns ErrFilterComplexityExceeded at <=0; And/Or branches build via sql.AndPredicates/OrPredicates and skip nil predicates (`for _, child := range lo.FromPtr(f.And) { if err := child.validateWithComplexity(maxDepth-1); err != nil { return err } }`)
**LIKE metacharacter escaping for Contains** — `Contains`/`Ncontains` are literal substring matches built through `ContainsPattern` -> `EscapeLikePattern` which escapes \ % _; never hand-build a LIKE pattern from user input (`return q.ILike(field, ContainsPattern(*f.Contains))`)
**In-memory Match mirrors SQL semantics** — `FilterString.Match`/`matches` evaluate the filter against a Go value (Contains is case-insensitive, Like/Ilike return ErrOperationNotSupported); nil or empty filter matches everything (`func (f *FilterString) Match(value string) (bool, error) { if f == nil || f.IsEmpty() { return true, nil } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | All filter types, the `Filter` interface, sentinel errors, shared helpers `validateSingleOperator`, `isEmptyFilter`, `collectStringValues`, and the LIKE helpers `EscapeLikePattern`/`ContainsPattern`/`ReverseContainsPattern` | Select returns nil for empty filters and SelectWhereExpr returns "" — callers must guard before applying; ILIKE in Ent is hand-built via sql.P raw SQL, not a helper, so a new operator needs both Select and SelectWhereExpr branches kept in sync |
| `filter_test.go` | Table-driven parity tests asserting identical results from go-sqlbuilder (`wantExprSQL`/Args) and Ent (`wantEntSQL`/Args) plus escaping/validation cases | Any new operator/filter type must add both branches to the table and assert via assertValidationError (which checks models.IsGenericValidationError + errors.Is) |

## Anti-Patterns

- Adding an operator to Select but not SelectWhereExpr (or vice versa) — the two SQL emitters silently diverge
- Building LIKE patterns from raw user input without EscapeLikePattern/ContainsPattern
- Double-wrapping validation errors instead of returning raw sentinels from validateWithComplexity and wrapping once at the public entry
- Allowing multiple operator fields to be set on one filter (must trip validateSingleOperator)
- Skipping the depth budget on recursive $and/$or, enabling unbounded-complexity filters

## Decisions

- **Each filter emits both Ent predicates and go-sqlbuilder expressions** — Ent-backed adapters and raw-SQL (e.g. ClickHouse/meter query) paths both need the same filter, so one struct serves both query builders
- **Operator set uses Mongo-style $-prefixed JSON tags** — Matches the AIP-style query-parameter API contract surfaced by api/v3/filters and keeps request decoding mechanical

## Example: Apply a string filter to an Ent query, guarding empties

```
f := filter.FilterString{Contains: lo.ToPtr("needle")}
if err := f.Validate(); err != nil { return err }
if pred := f.Select("name"); pred != nil {
	query = query.Where(pred)
}
```

<!-- archie:ai-end -->
