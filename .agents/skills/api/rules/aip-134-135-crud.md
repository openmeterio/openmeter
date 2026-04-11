# AIP-134 / AIP-135 — CRUD request & response templates

References:

- https://kong-aip.netlify.app/aip/134/ (update)
- https://kong-aip.netlify.app/aip/135/ (delete)

Use the generic templates from `shared/request.tsp` and `shared/responses.tsp`. Do not define ad-hoc request/response types.

| Purpose         | Request template          | Response template                   | HTTP status |
| --------------- | ------------------------- | ----------------------------------- | ----------- |
| Create (POST)   | `Shared.CreateRequest<T>` | `Shared.CreateResponse<T>`          | 201         |
| Upsert (PUT)    | `Shared.UpsertRequest<T>` | `Shared.UpsertResponse<T>`          | 200/201     |
| Update (PATCH)  | `Shared.UpdateRequest<T>` | `Shared.UpdateResponse<T>`          | 200         |
| Get (GET)       | —                         | `Shared.GetResponse<T>`             | 200         |
| Delete (DELETE) | —                         | `Shared.DeleteResponse`             | 204         |
| Page list       | —                         | `Shared.PagePaginatedResponse<T>`   | 200         |
| Cursor list     | —                         | `Shared.CursorPaginatedResponse<T>` | 200         |

## AIP-134 update rules

Both `PATCH` and `PUT` are **required** for all entities (mandate introduced 2025-04-08). The sole exception: `PATCH` may be omitted when the full entity representation is needed to validate an update.

### PATCH (partial update)

- Implements JSON Merge Patch (RFC 7396) with **mandatory recursive patching** of nested objects.
- Operation ID: `update-<entity>` (kebab-case).
- Rejects requests with `Content-Type` other than `application/json` with `400 Bad Request`.
- Rejects unknown fields and read-only fields with `400 Bad Request`, naming them in `invalid_parameters`.
- Null-value semantics:
  - For non-required properties: removes the property
  - For schema-less object properties: removes the property
  - For required-nullable properties: sets them to null
  - For required non-nullable properties: `400 Bad Request`

### PUT (upsert)

- Returns `201 Created` when creating an entity, `200 OK` when replacing an existing one.
- Operation ID: `upsert-<entity>` or `update-<entity>` (kebab-case).
- **Creation via PUT only works when the entity uses customer-supplied IDs** (unique within the organization or parent scope, not globally).
- For entities with system-generated globally-unique IDs, PUT only supports **replacement**; missing entities must return `404 Not Found`.

## AIP-135 delete rules

- DELETE returns `204 No Content` on success.
- **No request body is accepted**; all parameters go in the URL path, not the query string.
- Return `403 Forbidden` before `404 Not Found` — check permissions before existence to prevent resource enumeration.
- Soft-deleted resources return `404 Not Found` on subsequent DELETE calls.
- Cascading deletes:
  - If no protected resources would be affected, proceed as a normal DELETE.
  - If protected resources would be affected and `?force=true` is **not** supplied, return `400 Bad Request` listing the affected resource types in the error detail.
  - If `?force=true` is supplied, delete all child resources and associations, returning `204`.
