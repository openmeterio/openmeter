# filter

<!-- archie:ai-start -->

> Provides typed, composable query filters (string, integer, float, time, boolean, ULID) that compile to both Ent selector predicates and go-sqlbuilder WHERE expressions. The primary constraint is that every filter type implements the Filter interface and enforces single-operator-per-node validation.

## Patterns

**Filter interface dual output** — Every filter type implements both Select(field string) func(*sql.Selector) for Ent queries and SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string for ClickHouse/raw SQL. New filter types must implement both. (`predicate := myFilter.Select("name")  // -> func(*sql.Selector)
expr := myFilter.SelectWhereExpr("name", q)  // -> string`)
**Single-operator validation** — validateSingleOperator(f) uses reflection to count non-nil fields; Validate() wraps the result in models.NewNillableGenericValidationError. Public Validate/ValidateWithComplexity always wrap exactly once. (`if err := f.Validate(); err != nil { /* err is models.GenericValidationError */ }`)
**Recursive And/Or with depth limit** — Each filter type carries *[]FilterT And and *[]FilterT Or fields. validateWithComplexity(maxDepth) recurses and returns ErrFilterComplexityExceeded when maxDepth <= 0. (`if err := f.ValidateWithComplexity(3); errors.Is(err, filter.ErrFilterComplexityExceeded) { ... }`)
**IsEmpty guards nil return from Select** — Select returns nil for empty filters. Callers and implementations must check IsEmpty or nil before applying the predicate to an Ent selector. (`if p := f.Select("field"); p != nil { query.Where(p) }`)
**Contains uses ILIKE with escaped LIKE metacharacters** — $contains and $ncontains wrap the value with EscapeLikePattern + ContainsPattern before passing to ILIKE/NOT ILIKE. Raw $like/$ilike operators pass values unescaped. (`// Contains auto-escapes: filter.FilterString{Contains: lo.ToPtr("50%")} -> ILIKE '%50\%%'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | All filter struct types plus the Filter interface. EscapeLikePattern, ContainsPattern, and ReverseContainsPattern are the LIKE-safety helpers. | FilterString has both $like (caller-controlled pattern) and $contains (auto-escaped plain substring). Do not double-escape by passing an already-percent-wrapped value to $contains. |
| `filter_test.go` | Dual-path tests: SelectWhereExpr (go-sqlbuilder) and Select (Ent) branches are validated side by side for every operator variant. | Test SQL strings differ between backends (? vs $1 placeholders, quoted identifiers). Use newSelectBuilder() for Ent assertions. |

## Anti-Patterns

- Setting multiple operator fields on the same filter node (Validate returns ErrFilterMultipleOperators)
- Applying Select() predicate without nil check — empty filter returns nil and will panic on the selector
- Passing pre-escaped LIKE patterns to $contains/$ncontains — they will be double-escaped
- Adding a new filter type without implementing both Select and SelectWhereExpr
- Wrapping Validate() result in a second models.GenericValidationError — validateWithComplexity returns raw sentinels intentionally

## Decisions

- **Dual output (Ent selector + go-sqlbuilder expression) on a single Filter type.** — Domain adapters use Ent for Postgres and go-sqlbuilder for ClickHouse; one filter struct drives both backends without separate translation layers.
- **Validation wraps errors in models.GenericValidationError exactly once at the public boundary.** — Domain services call Validate() and forward the result directly to HTTP encoders that map GenericValidationError to 400; double-wrapping would break errors.Is checks.

## Example: Apply a validated FilterString to an Ent query and a ClickHouse sqlbuilder query

```
import (
    "github.com/openmeterio/openmeter/pkg/filter"
    "github.com/samber/lo"
)

f := filter.FilterString{Contains: lo.ToPtr("acme")}
if err := f.Validate(); err != nil {
    return nil, err
}

// Ent path:
if pred := f.Select("display_name"); pred != nil {
    query = query.Where(pred)
}

// ...
```

<!-- archie:ai-end -->
