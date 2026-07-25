---
name: samber-lo
description: Use when writing or refactoring Go collection/pointer helper code in OpenMeter, especially when choosing between standard library slices/maps helpers and github.com/samber/lo for cloning, copying, equality, sorting, pointer literals, slice-to-map transforms, map keys/values, mapping, filtering, grouping, uniqueness, set-like conversions, and map entry transformations.
user-invocable: true
argument-hint: "[collection transformation or lo helper question]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob
---

# samber/lo

Use standard library collection helpers first when they express the operation directly, and use `github.com/samber/lo` for small, local collection transformations when it makes the intent clearer than a hand-written loop. OpenMeter pins `github.com/samber/lo v1.53.0` in `go.mod`.

## Imports

```go
import "maps"
import "slices"
import "github.com/samber/lo"
```

Do not use dot imports. Do not reach for `lo/parallel`, `lo/mutable`, or `lo/it` unless the surrounding package already uses that subpackage or the task explicitly needs it.

## Standard Library First

Prefer `slices` for operations the standard library already names clearly:

- `slices.Clone(s)` for defensive slice copies instead of `append([]T(nil), s...)`.
- `slices.Contains(s, v)` for membership checks on short or already-available slices.
- `slices.Sort`, `slices.SortFunc`, and `slices.SortStableFunc` before comparing, logging, serializing, or asserting on order.
- `slices.Equal` and `slices.EqualFunc` for equality checks.
- `slices.Concat(a, b, c)` for concatenating known slices.
- `slices.Compact` after sorting when normalizing a slice.

Prefer `maps` for direct map operations:

- `maps.Clone(m)` for defensive copies.
- `maps.Copy(dst, src)` when merging maps.
- `maps.Equal` and `maps.EqualFunc` for equality checks.
- `maps.Keys(m)` and `maps.Values(m)` when the iterator result is acceptable; collect or sort when a slice is required or order matters.

Prefer `lo.ToPtr(...)`, `lo.FromPtr(...)`, and `lo.FromPtrOr(...)` for pointer literals and pointer defaults. Avoid local wrappers such as `ptr`, `loPtr`, `must`, or `loMust`.

## Common Helpers

For slice transforms:

- `lo.Map(items, func(item T, index int) R)` converts `[]T` to `[]R`.
- `lo.Filter(items, func(item T, index int) bool)` keeps matching items.
- `lo.FilterMap(items, func(item T, index int) (R, bool))` filters and maps in one pass.
- `lo.Uniq(items)` and `lo.UniqBy(items, func(item T) K)` remove duplicates.
- `lo.GroupBy(items, func(item T) K)` returns `map[K][]T`.
- `lo.GroupByMap(items, func(item T) (K, V))` returns `map[K][]V`.

For slice-to-map work:

- `lo.KeyBy(items, func(item T) K)` returns `map[K]T`.
- `lo.SliceToMap(items, func(item T) (K, V))` returns `map[K]V`; `lo.Associate` is the same operation, but prefer `SliceToMap` for readability.
- `lo.FilterSliceToMap(items, func(item T) (K, V, bool))` returns `map[K]V` while skipping items.
- `lo.Keyify(items)` returns `map[T]struct{}` for set-like membership.

For map work:

- `lo.Keys(m)` and `lo.Values(m)` return slices.
- `lo.UniqKeys(m1, m2)` and `lo.UniqValues(m1, m2)` combine maps while removing duplicates.
- `lo.MapKeys(m, func(value V, key K) R)` changes key type while preserving values.
- `lo.MapValues(m, func(value V, key K) R)` changes values while preserving keys.
- `lo.MapEntries(m, func(key K, value V) (K2, V2))` changes both keys and values.
- `lo.MapToSlice(m, func(key K, value V) R)` converts a map to a slice.
- `lo.FilterMapToSlice(m, func(key K, value V) (R, bool))` filters and maps a map into a slice.
- `lo.PickBy`, `lo.OmitBy`, `lo.FilterKeys`, and `lo.FilterValues` are useful when only part of a map is needed.

Use `*Err` variants such as `MapErr`, `KeyByErr`, `GroupByMapErr`, `MapValuesErr`, or `MapToSliceErr` when the callback can fail and the first error should stop the transform.

For slice-wide invariants where the exact offending element is not important, prefer collecting distinct values with `lo.Map` plus `lo.Uniq` and validating cardinality over stateful "first seen value" loops.

## Correctness Notes

- Duplicate keys in `KeyBy`, `SliceToMap`, `MapKeys`, and `MapEntries` overwrite earlier entries; the last value wins.
- Map iteration order is not stable. Sort keys or results before comparing, logging, serializing, or asserting on order.
- Callback argument order differs by helper: slice helpers usually pass `(item, index)`, map helpers usually pass `(value, key)` for `MapKeys`/`MapValues` and `(key, value)` for `MapEntries`/`MapToSlice`.
- Prefer a plain `for` loop when the transform has branching business rules, context-aware calls, transaction-sensitive operations, side effects, or multi-step error handling.
- Keep code inline for tiny transformations; do not add pass-through wrappers around `lo` helpers unless the wrapper name captures a domain rule that is not obvious from the helper call.

## Examples

```go
ids := lo.Map(customers, func(customer customer.Customer, _ int) string {
    return customer.ID
})

byID := lo.KeyBy(customers, func(customer customer.Customer) string {
    return customer.ID
})

namesByID := lo.SliceToMap(customers, func(customer customer.Customer) (string, string) {
    return customer.ID, customer.Name
})

activeNamesByID := lo.FilterSliceToMap(customers, func(customer customer.Customer) (string, string, bool) {
    return customer.ID, customer.Name, customer.Active
})

externalIDs := lo.Keys(externalIDByCustomerID)

apiByID := lo.MapValues(customerByID, func(customer customer.Customer, _ string) api.Customer {
    return mapCustomerToAPI(customer)
})
```
