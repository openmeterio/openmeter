# convert

<!-- archie:ai-start -->

> Nil-safe generic pointer and type conversion helpers used throughout the codebase to avoid repetitive nil-guard boilerplate when mapping between domain types and API/Ent types.

## Patterns

**ToPointer for value-to-pointer** — Use convert.ToPointer[T](value) instead of taking the address of a local variable directly. (`convert.ToPointer("some-string")`)
**MapToPointer / SliceToPointer return nil on empty** — Both functions return nil (not a pointer to empty collection) when the collection is empty. Use when an absent collection must differ from an empty one in JSON serialization. (`convert.MapToPointer(myMap) // returns nil if len==0`)
**SafeDeRef for nil-safe pointer transformation** — SafeDeRef applies a mapping function to a pointer only if non-nil, otherwise returns nil. Replaces repetitive if ptr != nil checks in converters. (`convert.SafeDeRef(req.StartTime, func(t time.Time) *time.Time { return convert.ToPointer(t.UTC()) })`)
**ToStringLike for type-aliased string conversions** — Converts *Source to *Dest where both are ~string types, avoiding unsafe casts. (`convert.ToStringLike[api.CurrencyCode, currencyx.Code](req.Currency)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ptr.go` | All pointer/type helpers including DerefHeaderPtr for *map and *slice dereferencing. | lo.EmptyableToPtr is explicitly avoided for maps/slices — use MapToPointer/SliceToPointer instead. |
| `time.go` | TimePtrIn converts *time.Time to a given location, nil-safe. | Returns a new pointer; does not mutate in place. |

## Anti-Patterns

- Using lo.EmptyableToPtr on maps or slices — it does not treat empty collections as nil
- Adding domain-specific conversion logic here — this package is for generic pointer/type mechanics only

<!-- archie:ai-end -->
