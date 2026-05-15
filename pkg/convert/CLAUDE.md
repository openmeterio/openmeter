# convert

<!-- archie:ai-start -->

> Nil-safe generic pointer and type conversion helpers used throughout the codebase to eliminate repetitive nil-guard boilerplate when mapping between domain types, API types, and Ent-generated types.

## Patterns

**ToPointer for value-to-pointer** — Use convert.ToPointer[T](value) instead of taking the address of a local variable directly. (`convert.ToPointer("some-string")`)
**MapToPointer / SliceToPointer return nil on empty** — Both return nil (not a pointer to empty collection) when the input is empty. Prefer over lo.EmptyableToPtr for maps and slices. (`convert.MapToPointer(myMap) // returns nil if len==0`)
**SafeDeRef for nil-safe pointer transformation** — Applies a mapping function only if the pointer is non-nil; returns nil otherwise. Replaces repetitive if ptr != nil blocks in converter files. (`convert.SafeDeRef(req.StartTime, func(t time.Time) *time.Time { return convert.ToPointer(t.UTC()) })`)
**ToStringLike for type-aliased string conversions** — Converts *Source to *Dest where both are ~string kind, avoiding unsafe casts. (`convert.ToStringLike[api.CurrencyCode, currencyx.Code](req.Currency)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ptr.go` | All pointer/type helpers including DerefHeaderPtr for *map and *slice dereferencing. Also StringerPtrToStringPtr for fmt.Stringer types. | lo.EmptyableToPtr is explicitly avoided for maps/slices — always use MapToPointer/SliceToPointer instead to get nil-on-empty semantics. |
| `time.go` | TimePtrIn converts *time.Time to a given location, nil-safe. Returns a new pointer; does not mutate in place. | Returns nil when input is nil — callers must not assume a non-nil result. |

## Anti-Patterns

- Using lo.EmptyableToPtr on maps or slices — it does not treat empty collections as nil
- Adding domain-specific conversion logic here — this package is for generic pointer/type mechanics only
- Taking the address of a local variable instead of convert.ToPointer when the intent is nil-safety

<!-- archie:ai-end -->
