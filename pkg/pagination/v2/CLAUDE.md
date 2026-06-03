# v2

<!-- archie:ai-start -->

> Generic cursor-based pagination library providing Cursor (time+ID composite, base64-encoded), Result[T], Paginator[T] interface, and CollectAll. All domain list queries needing stable keyset pagination must use these types — never offset-based pagination.

## Patterns

**Item interface for cursor derivation** — Any type passed to NewResult[T] must implement Item (Cursor() Cursor); NewResult derives the next-page cursor from the last element's Cursor() automatically. (`func (e MyEntity) Cursor() pagination.Cursor { return pagination.NewCursor(e.CreatedAt, e.ID) }`)
**Cursor encoding: base64(RFC3339,ID)** — Cursors are opaque. Internal format is base64('<RFC3339 UTC>,<ID>'). Always use NewCursor/Encode/DecodeCursor; time is normalised to UTC; SplitN(limit=2) allows commas in IDs. (`c := pagination.NewCursor(t, id); encoded := c.Encode(); decoded, err := pagination.DecodeCursor(encoded)`)
**Paginator[T] via NewPaginator** — Wrap any page-fetch function as a Paginator[T]; the function receives a *Cursor (nil on first call) and returns Result[T] with Items and NextCursor. (`p := pagination.NewPaginator[MyType](func(ctx context.Context, c *pagination.Cursor) (pagination.Result[MyType], error) { ... })`)
**CollectAll with MAX_SAFE_ITER cap** — CollectAll iterates until NextCursor is nil, capped at MAX_SAFE_ITER (10_000). Returns partial results alongside any mid-iteration error. (`items, err := pagination.CollectAll[MyType](ctx, paginator, nil)`)
**nil NextCursor signals last page** — Result.NextCursor == nil means no further pages; NewResult sets it nil when the items slice is empty. Check before issuing another request. (`if res.NextCursor == nil { break }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pagination.go` | Item interface (Cursor() Cursor) and Paginator[T] interface with NewPaginator wrapping a closure. | Adding methods to Item breaks all implementors; keep it to Cursor() Cursor only. |
| `cursor.go` | Cursor value type with base64 encode/decode, MarshalText/UnmarshalText, Validate, EncodePtr. Time is always UTC. | Do not change cursorDelimiter or encoding format — client-stored cursors break. SplitN(limit=2) intentionally allows commas in IDs. |
| `result.go` | Result[T] and NewResult[T Item]; NextCursor derived from the last item's Cursor(). | NewResult requires T to implement Item. When the caller controls the next cursor manually, build Result[T] directly without NewResult. |
| `collect.go` | CollectAll iterates a Paginator until exhausted or MAX_SAFE_ITER (10_000). | Silently stops at 10_000 without error; callers must handle both return values — partial results are returned alongside errors. |

## Anti-Patterns

- Implementing offset-based pagination — this package is cursor-only.
- Manually constructing the cursor string instead of NewCursor/Encode.
- Calling NewResult[T] without T implementing Item.
- Ignoring the (items, error) dual return from CollectAll — partial results are intentional.
- Storing decoded Cursor fields externally and reconstructing — treat Cursor as opaque; re-encode via Encode().

## Decisions

- **Cursor encodes both time and ID (composite key)** — Time alone is not unique across rows; the ID disambiguates ties, enabling stable keyset pagination when records share a timestamp.
- **MAX_SAFE_ITER cap of 10_000 in CollectAll** — Prevents runaway memory if a Paginator never returns nil NextCursor; tests assert the cap is honoured.
- **Paginator[T] as an interface wrapping a function via NewPaginator** — Lets test paginators be closures without a named type, keeping test code concise.

## Example: Domain list method returning paginated results with a next cursor

```
import (
    "context"
    pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type MyEntity struct { ID string; CreatedAt time.Time }
func (e MyEntity) Cursor() pagination.Cursor { return pagination.NewCursor(e.CreatedAt, e.ID) }

func (s *service) List(ctx context.Context, cursor *pagination.Cursor) (pagination.Result[MyEntity], error) {
    items := s.fetch(ctx, cursor)
    return pagination.NewResult(items), nil
}
```

<!-- archie:ai-end -->
