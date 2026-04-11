# AIP-132 — List endpoints

Reference: https://kong-aip.netlify.app/aip/132/

## Method, URL, and response shape

- List endpoints use `GET` on the entity root (e.g., `GET /meters`).
- Responses must contain a `data` key holding the array of requested entities — this comes from the pagination response templates (`Shared.PagePaginatedResponse<T>`, etc.).
- **Trailing slash returns `404`** — `GET /meters/` must 404, `GET /meters` must 200.
- List endpoints must follow the pagination AIP (see `aip-158-pagination.md`) unless the resource will provably never need pagination.

## Default sort order is mandatory and deterministic

AIP-132 requires a default sort order: *"Multiple requests to retrieve the same list must result in the same ordering of items."* Pick a field that is both stable and unique enough to produce a total order.

**OpenMeter convention** (not AIP-132 itself): prefer `name asc`, `created_at asc`, or `updated_at desc` as the default. Avoid UUIDs as the sort key — they are unique but randomly ordered, which makes the default sort feel arbitrary to clients. Always pair a non-unique sort column (like `name`) with a tie-breaker (like `id`) to preserve determinism.

## `sort` query parameter

Expose sorting via `@query(#{ name: "sort" }) sort?: Common.SortQuery`.

Sort values are a **comma-delimited list of attributes with optional `asc`/`desc` suffixes**, with the following syntax rules (from AIP-132):

- Ascending is the default — the `asc` suffix is optional
- Descending requires the `desc` suffix, separated by a space
- Multiple attributes sort left-to-right
- Extra whitespace around delimiters is insignificant
- JSONPath dot notation supports nested attributes (e.g., `foo.bar`)

**AIP example:** `?sort=foo,bar desc,foo.baz asc`

Sortable attributes should include all filterable attributes (so clients can sort on anything they can filter on). If the server cannot execute a particular sort expression, return `400 Bad Request`.
