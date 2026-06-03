# slicesx

<!-- archie:ai-start -->

> Thin extension layer on top of samber/lo providing error-aware map, diff, and early-exit iteration helpers that lo omits; all functions are generic and pure with no side effects or domain imports.

## Patterns

**MapWithErr accumulates all errors** — MapWithErr uses errors.Join to collect errors from every element — it does NOT short-circuit. The result slice omits erroring elements; count may be less than input. (`results, err := slicesx.MapWithErr(items, func(i Item) (Out, error) { return convert(i) })`)
**Nil-in nil-out contract** — Map and MapWithErr return nil (not empty slice) for nil input. Callers needing a non-nil empty slice must initialize the input first. (`slicesx.Map(nil, f) // returns nil, not []S{}`)
**NewDiff argument order: base then new** — NewDiff(base, new) wraps lo.Difference — additions are in base not in new; removals are in new not in base. Swapping arguments silently inverts semantics. (`d := slicesx.NewDiff(oldIDs, newIDs); if d.InAdditions(id) { /* id was removed from new set */ }`)
**ForEachUntilWithErr: break and error are independent** — Callback returning (true, nil) breaks without error; (false, err) stops with error. Both stop iteration but are semantically distinct. (`slicesx.ForEachUntilWithErr(items, func(v Item, i int) (bool, error) { return v.Done, nil })`)
**Prefer lo for covered operations** — Filter, Contains, Uniq, Keys already exist in samber/lo and must not be re-implemented. Add here only when lo lacks the pattern. (`// Use lo.Filter, lo.Contains — not a new slicesx.Filter`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Diff[T,S] value type wrapping lo.Difference with O(1) index maps for InAdditions/InRemovals/Has plus HasChanged. | NewDiff arg order: first is base (additions), second is new (removals). Swapping inverts semantics. |
| `map.go` | Map (no-error transform) and MapWithErr (error-accumulating transform). | MapWithErr skips erroring elements and continues — result may be shorter than input; error is errors.Join of all failures. |
| `empty.go` | EmptyAsNil normalises zero-length slices to nil for struct equality assertions. | Intended for test code only — using in production can hide unintended empty-slice semantics. |
| `each.go` | ForEachUntilWithErr iterates with early-exit via (breaks bool, err error) from callback. | breaks and err are independent — (true, nil) breaks cleanly; never assume breaks==true implies no error. |
| `groupby.go` | UniqueGroupBy groups and asserts uniqueness — returns (nil, false) if any key maps to more than one element. | Returns (nil, false) on collision rather than an error — callers must check the bool. |

## Anti-Patterns

- Duplicating lo functions (Filter, Contains, Uniq) here — use lo directly.
- Using EmptyAsNil in production code — it is for test assertion normalisation only.
- Adding stateful or side-effectful helpers — all functions must be pure transforms.
- Assuming MapWithErr returns same-length output when errors occur — erroring elements are skipped.

## Decisions

- **Wrap lo rather than replace it.** — lo covers ~90% of slice operations; slicesx fills only the gaps (error-returning maps, O(1) diff, early-exit iteration) without duplicating the rest.

## Example: Convert domain entities to API types, collecting all conversion errors

```
import "github.com/openmeterio/openmeter/pkg/slicesx"

apiItems, err := slicesx.MapWithErr(domainItems, func(item domain.Entity) (api.Entity, error) {
    return toAPIEntity(item)
})
if err != nil { return nil, err }
```

<!-- archie:ai-end -->
