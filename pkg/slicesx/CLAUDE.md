# slicesx

<!-- archie:ai-start -->

> Thin extension layer on top of samber/lo providing error-aware map, diff, and iteration helpers that lo omits; all functions are generic and pure (no side effects, no domain imports).

## Patterns

**MapWithErr collects all errors before returning** — MapWithErr uses errors.Join to accumulate errors across all elements and returns nil slice + joined error if any mapping fails; it does NOT short-circuit on first error. (`results, err := slicesx.MapWithErr(items, func(i Item) (Out, error) { return convert(i) })`)
**Nil-in nil-out contract** — Map and MapWithErr both return nil (not an empty slice) when given a nil input — callers relying on non-nil output must pass an initialized slice. (`slicesx.Map(nil, f) // returns nil, not []S{}`)
**NewDiff wraps lo.Difference with O(1) membership lookup** — NewDiff(base, new) builds index maps for both additions and removals so Has/InAdditions/InRemovals are O(1). (`d := slicesx.NewDiff(oldIDs, newIDs); if d.InAdditions(id) { ... }`)
**Prefer lo for standard operations** — Functions that lo already covers (Filter, Contains, Keys, etc.) should NOT be re-implemented here; add to slicesx only for patterns lo explicitly lacks. (`// Use lo.Filter, not a custom slicesx.Filter`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Diff[T,S] value type for comparing two slices; additions = items in base not in new, removals = items in new not in base (mirrors lo.Difference semantics). | Diff argument order: NewDiff(base, new) — swapping them inverts additions/removals. |
| `map.go` | Map (no-error) and MapWithErr (error-accumulating) generic slice transforms. | MapWithErr skips erroring elements and continues — the result slice omits them; check for this when order or count matters. |
| `empty.go` | EmptyAsNil normalises zero-length slices to nil for struct equality in tests. | Only use in tests or assertion helpers; calling it in production can hide unintended empty-slice semantics. |
| `each.go` | ForEachUntilWithErr iterates with early-exit support via a bool return from the callback. | Callback returning (true, nil) breaks the loop without error; returning (false, err) stops with error — these are independent signals. |

## Anti-Patterns

- Duplicating lo functions (Filter, Contains, Uniq) here — use lo directly
- Using EmptyAsNil in production code paths — it is intended for test assertion normalisation only
- Adding stateful or side-effectful helpers — all functions here must be pure transforms

## Decisions

- **Wrap lo rather than replace it** — lo covers ~90% of slice operations; slicesx fills the gaps (error-returning maps, diff with index, early-exit iteration) without duplicating the rest.

## Example: Convert a slice of domain entities to API types, collecting all conversion errors

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
