# v2

<!-- archie:ai-start -->

> Cursor-based (keyset) pagination primitives. Provides an opaque base64 Cursor (time+ID), a generic Paginator[T] interface backed by a closure, a Result[T] page wrapper, and CollectAll to drain a paginator across pages. This is the v2 cursor pagination model distinct from the older offset/limit pagination in pkg/pagination.

## Patterns

**Cursor is always UTC-normalized** — NewCursor stores Time as t.UTC(); Encode re-applies .UTC() before formatting. Two cursors for the same instant in different zones must encode identically. (`cursor := NewCursor(nyTime, "id") // cursor.Time == nyTime.UTC()`)
**Cursor wire format is base64(<RFC3339 time>,<ID>)** — Encode/Decode use cursorDelimiter "," and SplitN(...,2) so the ID may itself contain commas. Never hand-construct the encoded string; round-trip via Encode/DecodeCursor. (`encoded := c.Encode(); c2, err := DecodeCursor(encoded)`)
**Cursor implements encoding.Text(Un)Marshaler** — MarshalText delegates to Encode, UnmarshalText to the decode path. EncodePtr returns nil for a nil receiver so it maps cleanly to optional API fields. (`nextCursor := result.NextCursor.EncodePtr() // *string, nil-safe`)
**Paginate via NewPaginator closure, not a custom struct** — Paginator[T] is a one-method interface; NewPaginator wraps a func(ctx, *Cursor) (Result[T], error). Implement pagination by supplying the closure rather than defining new Paginator types. (`p := NewPaginator[int](func(ctx context.Context, cur *Cursor) (Result[int], error) { ... })`)
**NextCursor derived from last item's Cursor()** — NewResult requires T to implement Item (Cursor() Cursor) and sets NextCursor from the last item, or nil for an empty page. Items signal end-of-data by returning Result with NextCursor==nil. (`func (i TestItem) Cursor() Cursor { return NewCursor(i.CreatedAt, i.ID) }`)
**CollectAll is bounded and error-tolerant** — CollectAll loops at most MAX_SAFE_ITER (10_000) pages, copies the incoming cursor defensively, stops when NextCursor==nil, and on error returns the items gathered so far alongside the error (not nil). (`all, err := CollectAll[T](ctx, paginator, cursor) // partial 'all' usable even when err != nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `cursor.go` | Cursor{Time,ID} value type: NewCursor, Validate, Encode/EncodePtr, DecodeCursor, MarshalText/UnmarshalText. | Encode forces UTC and uses time.RFC3339 (second precision) — sub-second time is lost across a round trip. Validate rejects zero Time and empty ID. |
| `pagination.go` | Item interface (Cursor() Cursor), Paginator[T] interface, and NewPaginator closure constructor. | paginator[T] is unexported; the only construction path is NewPaginator. Don't add parallel Paginator implementations unless genuinely needed. |
| `result.go` | Result[T]{Items, NextCursor} page struct and NewResult constructor (T constrained to Item). | NewResult's NextCursor is the LAST item's own cursor — it is an inclusive keyset marker, not an exclusive offset. Empty input yields nil NextCursor. |
| `collect.go` | CollectAll generic drain loop with MAX_SAFE_ITER cap and defensive cursor copy. | An always-non-nil NextCursor paginator silently caps at 10_000 items with no error; ensure terminating conditions set NextCursor==nil. |

## Anti-Patterns

- Constructing or parsing the encoded cursor string manually instead of via Encode/DecodeCursor (breaks base64 + comma-in-ID handling).
- Storing or comparing Cursor.Time in a non-UTC zone — breaks encode stability and equality assertions.
- Relying on sub-second precision surviving Encode/Decode (RFC3339 truncates to seconds).
- Defining a new Paginator[T] struct instead of NewPaginator; or building offset-style pagination here (that belongs in the v1 pkg/pagination).
- Assuming CollectAll returns nil items on error — it returns the partially collected slice.

## Decisions

- **Cursor encodes time+ID and is UTC-normalized + base64.** — Keyset pagination needs a stable, opaque, zone-independent ordering token safe to pass over HTTP query params.
- **CollectAll returns gathered items together with any mid-stream error.** — Callers can use partial results and still surface the failure, rather than losing already-fetched pages.
- **Paginator is a closure-backed generic interface.** — Lets each call site supply its own page-fetch logic without a bespoke type, keeping the package a pure utility.

## Example: Build a paginator and drain all pages with CollectAll

```
p := pagination.NewPaginator[int](func(ctx context.Context, cur *pagination.Cursor) (pagination.Result[int], error) {
    page, next, err := fetchPage(ctx, cur)
    if err != nil {
        return pagination.Result[int]{}, err
    }
    return pagination.Result[int]{Items: page, NextCursor: next}, nil
})

all, err := pagination.CollectAll[int](ctx, p, nil)
if err != nil {
    // 'all' still holds items collected before the failure
    return all, err
}
```

<!-- archie:ai-end -->
