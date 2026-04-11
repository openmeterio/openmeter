# AIP-235 — Bulk delete

Reference: https://kong-aip.netlify.app/aip/235/

Bulk delete is an RPC-style `POST` at `/<entity>/bulk-delete` (not `DELETE`, since DELETE does not accept a body — see `aip-134-135-crud.md`).

## Request body

```json
{
  "data": [
    { "id": "..." },
    { "id": "..." }
  ]
}
```

- The request body is a top-level `data` array of objects, each with a required `id` field (**not** a flat `{ "ids": [...] }` array).
- `maxItems: 100` — up to 100 items per request.

## Two modes

### Transactional (all-or-nothing)

- Success → `204 No Content`
- If any target is missing or the caller lacks permission, the whole operation fails with `400 Bad Request`

### Non-transactional (partial success allowed)

- Always returns `207 Multi-Status`
- Response body is a list of per-item `{ id, status, description? }` objects
- `status` is `"success"` or `"failure"`; `description` is an optional explanation for failures

Pick transactional semantics when the caller must be able to assume "all or nothing". Pick non-transactional when the caller wants the backend to do as much as it can and report per-item outcomes.
