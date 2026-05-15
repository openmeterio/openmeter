# v2

<!-- archie:ai-start -->

> Generic cursor-based pagination library providing Cursor (time+ID composite, base64-encoded), Result[T], Paginator[T] interface, and CollectAll helper. All domain list queries needing stable keyset pagination must use these types — never offset-based pagination.

## Patterns

**Item interface for cursor derivation** — Any type passed to NewResult[T] must implement Item (single method: Cursor() Cursor). NewResult derives the next-page cursor from the last element's Cursor() call automatically. (`func (e MyEntity) Cursor() pagination.Cursor { return pagination.NewCursor(e.CreatedAt, e.ID) }`)
**Cursor encoding: base64(RFC3339,ID)** — Cursors are opaque to callers. Internal format is base64-encoded '<RFC3339 UTC>,<ID>'. Always use NewCursor/Encode/DecodeCursor — never construct the base64 string manually. Time is always normalised to UTC. SplitN with limit=2 allows commas in IDs. (`c := pagination.NewCursor(t, id); encoded := c.Encode(); decoded, err := pagination.DecodeCursor(encoded)`)
**Paginator[T] via NewPaginator** — Wrap any page-fetch function as a Paginator[T] using NewPaginator. The function receives a *Cursor (nil on first call) and returns Result[T] with Items and NextCursor. (`p := pagination.NewPaginator[MyType](func(ctx context.Context, c *pagination.Cursor) (pagination.Result[MyType], error) { ... })`)
**CollectAll with MAX_SAFE_ITER cap** — CollectAll iterates pages until NextCursor is nil, capped at MAX_SAFE_ITER (10_000) iterations. Returns partial results alongside any mid-iteration error. Never use for unbounded production loads without awareness of the cap. (`items, err := pagination.CollectAll[MyType](ctx, paginator, nil)`)
**nil NextCursor signals last page** — Result.NextCursor == nil means no further pages. NewResult sets NextCursor to nil when items slice is empty. Callers must check this before issuing another page request. (`if res.NextCursor == nil { break }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pagination.go` | Defines Item interface (Cursor() Cursor) and Paginator[T] interface with NewPaginator constructor wrapping a closure. | Adding methods to Item breaks all existing implementors; keep it to Cursor() Cursor only. |
| `cursor.go` | Cursor value type with base64 encode/decode, MarshalText/UnmarshalText, Validate, and EncodePtr helper. Time is always UTC. | Do not change cursorDelimiter or encoding format — existing encoded cursors stored by clients will break. SplitN limit=2 intentionally allows commas in IDs. |
| `result.go` | Result[T] struct and NewResult[T Item] constructor. NextCursor is derived from the last item's Cursor() method. | NewResult requires T to implement Item. When the caller controls the next cursor manually, build Result[T] directly without NewResult. |
| `collect.go` | CollectAll iterates a Paginator until exhausted or MAX_SAFE_ITER (10_000) reached. Returns (partialItems, err) on mid-iteration error. | Silently stops at 10_000 items without error. Callers must handle both return values — partial results are intentionally returned alongside errors. |

## Anti-Patterns

- Implementing offset-based pagination — this package is cursor-only; offset logic belongs elsewhere
- Manually constructing the cursor string (e.g. base64-encoding a custom format) instead of using NewCursor/Encode
- Calling NewResult[T] without T implementing Item — compile error, but watch for wrapper types that forget the interface
- Ignoring the (items, error) dual return from CollectAll — partial results are intentionally returned alongside errors
- Storing decoded Cursor fields (Time/ID) externally and reconstructing — treat Cursor as opaque; always re-encode via Encode()

## Decisions

- **Cursor encodes both time and ID (composite key)** — Time alone is not unique across rows; the ID disambiguates ties, enabling stable keyset pagination even when multiple records share the same timestamp.
- **MAX_SAFE_ITER cap of 10_000 in CollectAll** — Prevents runaway memory usage if a Paginator implementation never returns nil NextCursor. Tests explicitly assert the cap is honoured.
- **Paginator[T] as an interface wrapping a function via NewPaginator** — Allows test paginators to be constructed as closures without implementing a named type, keeping test code concise.

## Example: Implement a domain list method that returns paginated results with a next cursor

```
import (
	"context"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type MyEntity struct {
	ID        string
	CreatedAt time.Time
}

func (e MyEntity) Cursor() pagination.Cursor {
	return pagination.NewCursor(e.CreatedAt, e.ID)
}

func (s *service) List(ctx context.Context, cursor *pagination.Cursor) (pagination.Result[MyEntity], error) {
// ...
```

<!-- archie:ai-end -->
