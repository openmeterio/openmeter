---
name: samber-lo
description: Use when writing or refactoring Go code in OpenMeter that can use github.com/samber/lo helpers, especially trivial slice-to-map, map keys/values, mapping, filtering, grouping, uniqueness, set-like conversions, and map entry transformations.
user-invocable: true
argument-hint: "[collection transformation or lo helper question]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob
---

# samber/lo

Use `github.com/samber/lo` for small, local collection transformations when it makes the intent clearer than a hand-written loop. OpenMeter pins `github.com/samber/lo v1.53.0` in `go.mod`.

## Default Import

```go
import "github.com/samber/lo"
```

Do not use dot imports. Do not reach for `lo/parallel`, `lo/mutable`, or `lo/it` unless the surrounding package already uses that subpackage or the task explicitly needs it.

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
