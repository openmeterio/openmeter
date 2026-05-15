# response

<!-- archie:ai-start -->

> Reusable response envelope types for the v3 API: cursor-based and page-based pagination wrappers. All list endpoints must use one of these two structs to ensure consistent JSON shape across the API.

## Patterns

**Generic pagination wrappers via constructors** — Both CursorPaginationResponse[T] and PagePaginationResponse[T] are generic over item type T. Handlers call the constructor (NewCursorPaginationResponse / NewPagePaginationResponse) instead of building the struct manually. (`return response.NewCursorPaginationResponse(items, pageSize)`)
**Cursor derived from Item interface** — NewCursorPaginationResponse requires T to implement pagination.Item (exposes Cursor() method). First/Last cursors are set from items[0] and items[len-1]; Next is only set when a full page was returned. (`func NewCursorPaginationResponse[T pagination.Item](items []T, pageSize int) CursorPaginationResponse[T]`)
**nullable.Nullable for next/previous links** — CursorMetaPage.Next and Previous are nullable.Nullable[string] (not *string) so they serialize as explicit JSON null when absent rather than being omitted entirely. (`Next: nullable.NewNullNullable[string]() // produces JSON null, not omitted`)
**Next cursor gated on full page** — Next cursor is only populated when len(items) == pageSize; a partial page signals last page. Do not set Next manually from caller code. (`if len(items) == pageSize { result.Meta.Page.Next = nullable.NewNullableWithValue(lastItem.Cursor().Encode()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `cursorpagination.go` | Cursor-based pagination envelope; T must implement pagination.Item from pkg/pagination/v2. | Next cursor is only populated when len(items) == pageSize; a partial page signals last page. Do not set Next manually. |
| `pagepagination.go` | Offset/page-number pagination envelope; PageMetaPage.Total is optional (*int) for cases where total count is expensive. | Caller must populate PageMetaPage fully (Size, Number, Total) before passing to NewPagePaginationResponse — no defaults are inferred. |

## Anti-Patterns

- Returning a raw []T from a list endpoint instead of wrapping in CursorPaginationResponse or PagePaginationResponse
- Setting CursorMetaPage.Next to a non-null value when fewer items than pageSize were returned
- Using *string for Next/Previous fields — must use nullable.Nullable[string] to produce explicit JSON null
- Building CursorPaginationResponse struct manually instead of calling NewCursorPaginationResponse

## Decisions

- **Cursor encoded directly from the last item in the page rather than stored separately** — Avoids out-of-sync cursor state; the item itself is the canonical source of its own cursor position.
- **nullable.Nullable[string] used for Next/Previous instead of *string** — Produces explicit JSON null when absent rather than omitting the field, which is required by AIP pagination contracts for consistent client-side handling.

## Example: List handler wraps domain items in cursor pagination response

```
import (
    "github.com/openmeterio/openmeter/api/v3/response"
)

// T must implement pagination.Item (has Cursor() method)
return response.NewCursorPaginationResponse(items, pageSize), nil
```

<!-- archie:ai-end -->
