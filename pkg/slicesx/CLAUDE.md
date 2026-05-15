# slicesx

<!-- archie:ai-start -->

> Thin extension layer on top of samber/lo providing error-aware map, diff, and early-exit iteration helpers that lo omits; all functions are generic and pure with no side effects or domain imports.

## Patterns

**MapWithErr accumulates all errors before returning** — MapWithErr uses errors.Join to collect errors from every element — it does NOT short-circuit on first failure. The result slice omits erroring elements; count may be less than input. (`results, err := slicesx.MapWithErr(items, func(i Item) (Out, error) { return convert(i) })`)
**Nil-in nil-out contract** — Map and MapWithErr both return nil (not an empty slice) when given a nil input. Callers that require a non-nil empty slice must initialize the input first. (`slicesx.Map(nil, f) // returns nil, not []S{}`)
**NewDiff argument order: base then new** — NewDiff(base, new) wraps lo.Difference — additions are items in base not in new; removals are items in new not in base. Swapping arguments silently inverts additions and removals. (`d := slicesx.NewDiff(oldIDs, newIDs); if d.InAdditions(id) { /* id was removed from new set */ }`)
**ForEachUntilWithErr breaks on true, errors on err — these are independent signals** — Callback returning (true, nil) breaks the loop without error; (false, err) stops with error. Both stop iteration but are semantically distinct. (`slicesx.ForEachUntilWithErr(items, func(v Item, i int) (bool, error) { return v.Done, nil })`)
**Prefer lo for operations it already covers** — Functions like Filter, Contains, Uniq, Keys are already in samber/lo and must not be re-implemented in slicesx. Add here only when lo explicitly lacks the pattern. (`// Use lo.Filter, lo.Contains — not a new slicesx.Filter`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Diff[T,S] value type wrapping lo.Difference with O(1) index maps for InAdditions/InRemovals/Has and HasChanged convenience method. | NewDiff argument order: first arg is base (source of additions), second arg is new (source of removals). Swapping inverts semantics. |
| `map.go` | Map (no-error transform) and MapWithErr (error-accumulating transform) generic slice helpers. | MapWithErr skips erroring elements and continues — result slice may have fewer elements than input; returned error is errors.Join of all failures. |
| `empty.go` | EmptyAsNil normalises zero-length slices to nil for struct equality assertions. | Only intended for test code — using in production code can hide unintended empty-slice semantics. |
| `each.go` | ForEachUntilWithErr iterates with early-exit via (breaks bool, err error) return from callback. | breaks and err are independent — (true, nil) breaks cleanly; (false, err) stops with error. Never assume breaks==true implies no error. |
| `groupby.go` | UniqueGroupBy groups a collection and asserts uniqueness — returns (nil, false) if any key maps to more than one element. | Returns (nil, false) on collision rather than an error — callers must check the bool. |

## Anti-Patterns

- Duplicating lo functions (Filter, Contains, Uniq) here — use lo directly
- Using EmptyAsNil in production code — it is intended for test assertion normalisation only
- Adding stateful or side-effectful helpers — all functions must be pure transforms
- Assuming MapWithErr returns a result slice of the same length as input when errors occur — erroring elements are skipped

## Decisions

- **Wrap lo rather than replace it** — lo covers ~90% of slice operations; slicesx fills only the gaps (error-returning maps, diff with O(1) index, early-exit iteration) without duplicating the rest.

## Example: Convert a slice of domain entities to API types, collecting all conversion errors before returning

```
import "github.com/openmeterio/openmeter/pkg/slicesx"

apiItems, err := slicesx.MapWithErr(domainItems, func(item domain.Entity) (api.Entity, error) {
    return toAPIEntity(item)
})
if err != nil {
    return nil, err
}
```

<!-- archie:ai-end -->
