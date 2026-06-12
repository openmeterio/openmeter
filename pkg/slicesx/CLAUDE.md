# slicesx

<!-- archie:ai-start -->

> Generic slice helpers that complement github.com/samber/lo: error-aware mapping, ordered diffs, grouping, normalization, and nil/empty handling. One of the most widely imported pkg utilities across billing, subscription, entitlement, and api/v3.

## Patterns

**Nil-preserving Map** — Map[T,S] returns nil for nil input and otherwise a same-length slice; MapWithErr joins per-element errors with errors.Join and returns (nil, err) if any element fails. (`out, err := slicesx.MapWithErr(in, func(x T) (S, error) { ... })`)
**Diff as a queryable object** — NewDiff(base, new) wraps lo.Difference into a Diff with Additions/Removals/Changed/HasChanged plus O(1) membership via InAdditions/InRemovals/Has backed by internal maps. (`d := slicesx.NewDiff(base, next); if d.HasChanged() { ... }`)
**Fail-fast uniqueness grouping** — UniqueGroupBy returns (map, false) when any key maps to more than one element, instead of silently picking one. (`m, ok := slicesx.UniqueGroupBy(items, keyFn); if !ok { return errDuplicate }`)
**Normalize = sort + dedup** — Normalize[cmp.Ordered] clones, sorts, then slices.Compact; nil input stays nil. Used for stable comparable key lists. (`keys := slicesx.Normalize([]string{"b","a","a"}) // [a b]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `map.go` | Map and MapWithErr — the core mapping primitives. | Map preserves nil (returns nil, not []S{}); MapWithErr skips failed elements but ultimately returns nil slice plus joined error — never a partial slice. |
| `diff.go` | Diff[T,S] type + NewDiff built on lo.Difference and lo.SliceToMap. | Changed() does append(d.additions, d.removals...) which can mutate the additions backing array; treat the result as owned/throwaway. |
| `empty.go` | EmptyAsNil — collapses zero-length slices to nil, mainly for struct-equality in tests. | Intended for test equality; don't use to signal absence in production logic. |
| `each.go` | ForEachUntilWithErr — iterate with (breaks, err) control flow. | each_test.go contains a stray TestASDF smoke test unrelated to the package — harmless but noise. |
| `groupby.go` | UniqueGroupBy — group with single-element-per-key invariant. | Returns (nil, false) on collision; callers must check the bool. |
| `normalize.go` | Normalize — sort+compact for cmp.Ordered slices. | Compact only removes adjacent duplicates, hence the sort is required first. |
| `slice.go` | SliceToPtrSlice — []T to []*T taking addresses of the backing array. | Returned pointers alias the input slice's elements; mutating one mutates the other. |

## Anti-Patterns

- Returning []S{} instead of nil from Map-style helpers — breaks nil-preservation contract.
- Ignoring the bool from UniqueGroupBy and assuming the map is complete.
- Calling Normalize/Compact without sorting first for non-pre-sorted data.
- Reusing the slice returned by Diff.Changed() while still reading Additions().

## Decisions

- **Wraps lo rather than replacing it.** — diff.go and groupby.go import samber/lo; slicesx only adds the error-aware / invariant-checking variants lo lacks.
- **Nil input short-circuits in Map/MapWithErr.** — Keeps nil semantics so callers can distinguish 'no slice' from 'empty result'.

## Example: Map a slice of domain objects to DTOs, aggregating conversion errors

```
import "github.com/openmeterio/openmeter/pkg/slicesx"

lines, err := slicesx.MapWithErr(domainLines, func(l Line) (apiLine, error) {
    return toAPILine(l)
})
if err != nil { return nil, err }
```

<!-- archie:ai-end -->
