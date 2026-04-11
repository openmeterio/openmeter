# AIP-158 — Pagination

Reference: https://kong-aip.netlify.app/aip/158/

All collection endpoints must be paginated from inception — adding pagination later creates a backward-incompatible change. **Kong AIPs prioritize cursor pagination** for system-generated unbounded collections; offset/page-number pagination is reserved for exceptional cases with low-cardinality data.

## Cursor-based pagination (preferred)

- Spread `...Common.CursorPaginationQuery` into the operation.
- Parameters: `page[size]`, `page[after]`, `page[before]`.
- `after` and `before` are **mutually exclusive** — a request with both must return `400 Bad Request` (range pagination is not allowed).
- Return `Shared.CursorPaginatedResponse<T>` — response includes `meta.page.{size, next, previous, first?, last?}`.
  - `size` echoes the request parameter
  - `next` and `previous` URIs MUST be included when available
  - `first` and `last` URIs are optional, but if supported they should always be defined
- Cursors must be **opaque** to clients so the implementation can change without breaking them. The recommended encoding uses base64url with an XOR cipher to obscure cursor internals.
- `next`/`previous` URIs in the response must preserve all the caller's original query parameters except the pagination ones.

## Page-based pagination (offset-limit)

- Spread `...Common.PagePaginationQuery` into the operation.
- Parameters: `page[size]` and optional `page[number]`; default `number` is `1`.
- Return `Shared.PagePaginatedResponse<T>` — response includes `meta.page.{size, number, total?}`.
  - `size` and `number` both echo the request
  - `total` may be included

This style is acceptable only when you are sure the collection is small, bounded, and offset-friendly. Default to cursor pagination otherwise.

## OpenMeter opaque-offset pagination (Kong-common extension, not AIP-158)

`metadatas.yaml` also defines an additional offset-based pagination style **not described by AIP-158**:

- `PaginationOffset` — `?offset=<opaque>` query parameter (string, nullable)
- `PaginationSize` — `?size=<int>` query parameter (default 100, max 1000, min 1) — note the **flat** `size` name, not `page[size]`
- `PaginationNextResponse` — URI to the next page (may be null)
- `PaginationOffsetResponse` — the offset token to pass into the next list call

This style uses a flat query shape (`?size=10&offset=foo`) rather than the bracketed `page[...]` form required by AIP-158. **It is a local Kong-common addition, not an AIP-158-compliant style.** Prefer cursor or page-based pagination for new endpoints; reach for this opaque-offset form only when you need to match an existing contract that already exposes it.
