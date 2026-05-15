# filter

<!-- archie:ai-start -->

> Typed, composable query filters (string, integer, float, time, boolean, ULID) that compile to both Ent selector predicates (func(*sql.Selector)) and go-sqlbuilder WHERE expressions (string). Every filter type implements the Filter interface and enforces single-operator-per-node validation.

## Patterns

**Dual output — Ent selector and go-sqlbuilder expression** — Every filter type implements both Select(field string) func(*sql.Selector) for Ent/Postgres and SelectWhereExpr(field string, q *sqlbuilder.SelectBuilder) string for ClickHouse. New filter types must implement both methods. (`predicate := myFilter.Select("name")       // -> func(*sql.Selector)
expr := myFilter.SelectWhereExpr("name", q) // -> string`)
**Single-operator validation** — validateSingleOperator(f) uses reflection to count non-nil fields. Public Validate() wraps with models.NewNillableGenericValidationError exactly once — never double-wrap. (`if err := f.Validate(); err != nil { /* err is models.GenericValidationError */ }`)
**Recursive And/Or with depth limit** — Each filter type carries *[]FilterT And and *[]FilterT Or fields. ValidateWithComplexity(maxDepth) recurses and returns ErrFilterComplexityExceeded when maxDepth <= 0. (`if err := f.ValidateWithComplexity(3); errors.Is(err, filter.ErrFilterComplexityExceeded) { ... }`)
**IsEmpty / nil guard before applying predicate** — Select returns nil for empty filters. Always check IsEmpty() or nil before applying the predicate to an Ent selector. (`if p := f.Select("field"); p != nil { query.Where(p) }`)
**Contains uses ILIKE with auto-escaped metacharacters** — $contains and $ncontains wrap the value with EscapeLikePattern + ContainsPattern before passing to ILIKE/NOT ILIKE. Raw $like/$ilike pass values unescaped. (`// Contains auto-escapes: FilterString{Contains: lo.ToPtr("50%")} -> ILIKE '%50\%%'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | All filter struct types, the Filter interface, and LIKE-safety helpers (EscapeLikePattern, ContainsPattern, ReverseContainsPattern). | FilterString has both $like (caller-controlled pattern) and $contains (auto-escaped plain substring). Do not double-escape by passing an already-percent-wrapped value to $contains. FilterString.Match() returns ErrOperationNotSupported for Like/Nlike/Ilike/Nilike — these are SQL-only operators. |
| `filter_test.go` | Dual-path tests: SelectWhereExpr (go-sqlbuilder) and Select (Ent) branches validated side by side for every operator variant. | SQL placeholder syntax differs between backends (? vs $1). Use newSelectBuilder() (Ent/Postgres dialect) for Ent assertions; sqlbuilder.Select for go-sqlbuilder assertions. |

## Anti-Patterns

- Setting multiple operator fields on the same filter node — Validate returns ErrFilterMultipleOperators.
- Applying Select() predicate without nil check — empty filter returns nil and will panic on the selector.
- Passing pre-escaped LIKE patterns to $contains/$ncontains — they will be double-escaped.
- Adding a new filter type without implementing both Select and SelectWhereExpr.
- Wrapping Validate() result in a second models.GenericValidationError — validateWithComplexity returns raw sentinels intentionally; public Validate() wraps once.

## Decisions

- **Dual output (Ent selector + go-sqlbuilder expression) on a single Filter type.** — Domain adapters use Ent for Postgres and go-sqlbuilder for ClickHouse; one filter struct drives both backends without separate translation layers.
- **Validation wraps errors in models.GenericValidationError exactly once at the public boundary.** — Domain services call Validate() and forward the result directly to HTTP encoders that map GenericValidationError to 400; double-wrapping would break errors.Is checks.

## Example: Apply a validated FilterString to both an Ent query and a ClickHouse sqlbuilder query

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
