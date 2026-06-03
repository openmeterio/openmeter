# filter

<!-- archie:ai-start -->

> Typed, composable query filters (string, integer, float, time, boolean, ULID) that compile to both Ent selector predicates (func(*sql.Selector)) and go-sqlbuilder WHERE expressions (string). Every filter type implements the Filter interface and enforces single-operator-per-node validation.

## Patterns

**Dual output — Ent selector and go-sqlbuilder expression** — Every filter implements Select(field) func(*sql.Selector) for Ent/Postgres and SelectWhereExpr(field, q) string for ClickHouse; new types must implement both. (`predicate := myFilter.Select("name")
expr := myFilter.SelectWhereExpr("name", q)`)
**Single-operator validation** — validateSingleOperator(f) uses reflection to count non-nil fields; public Validate() wraps with models.NewNillableGenericValidationError exactly once — never double-wrap. (`if err := f.Validate(); err != nil { /* err is models.GenericValidationError */ }`)
**Recursive And/Or with depth limit** — Each type carries *[]FilterT And and Or fields; ValidateWithComplexity(maxDepth) recurses and returns ErrFilterComplexityExceeded when maxDepth <= 0. (`if err := f.ValidateWithComplexity(3); errors.Is(err, filter.ErrFilterComplexityExceeded) { ... }`)
**IsEmpty / nil guard before applying predicate** — Select returns nil for empty filters — check IsEmpty() or nil before applying to an Ent selector. (`if p := f.Select("field"); p != nil { query.Where(p) }`)
**Contains uses ILIKE with auto-escaped metacharacters** — $contains/$ncontains wrap the value with EscapeLikePattern + ContainsPattern before ILIKE; raw $like/$ilike pass values unescaped. (`// FilterString{Contains: lo.ToPtr("50%")} -> ILIKE '%50\%%'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | All filter struct types, the Filter interface, LIKE-safety helpers (EscapeLikePattern, ContainsPattern, ReverseContainsPattern). | $like is caller-controlled, $contains is auto-escaped — do not pass an already-percent-wrapped value to $contains; Match() returns ErrOperationNotSupported for Like/Nlike/Ilike/Nilike (SQL-only). |
| `filter_test.go` | Dual-path tests validating SelectWhereExpr and Select branches for every operator. | SQL placeholder syntax differs (? vs $1); use newSelectBuilder() for Ent, sqlbuilder.Select for go-sqlbuilder. |

## Anti-Patterns

- Setting multiple operator fields on the same node — Validate returns ErrFilterMultipleOperators.
- Applying Select() without a nil check — empty filter returns nil and panics on the selector.
- Passing pre-escaped LIKE patterns to $contains/$ncontains — double-escaping.
- Adding a new filter type without both Select and SelectWhereExpr.
- Wrapping Validate() result in a second GenericValidationError — public Validate() wraps once.

## Decisions

- **Dual output (Ent selector + go-sqlbuilder expression) on a single Filter type.** — Adapters use Ent for Postgres and go-sqlbuilder for ClickHouse; one filter struct drives both backends without separate translation layers.
- **Validation wraps in GenericValidationError exactly once at the public boundary.** — Services forward Validate() results to HTTP encoders mapping GenericValidationError to 400; double-wrapping breaks errors.Is checks.

## Example: Apply a validated FilterString to both Ent and ClickHouse queries

```
import (
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/samber/lo"
)
f := filter.FilterString{Contains: lo.ToPtr("acme")}
if err := f.Validate(); err != nil { return nil, err }
if pred := f.Select("display_name"); pred != nil { query = query.Where(pred) }
expr := f.SelectWhereExpr("display_name", sb)
```

<!-- archie:ai-end -->
