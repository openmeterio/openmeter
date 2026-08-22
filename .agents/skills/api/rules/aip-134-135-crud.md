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
- Rejects requests with `Content-Type` other than `application/json` with `400 Bad Request` (per AIP-134; note that AIP-137 prescribes `415 Unsupported Media Type` for content-type mismatches — prefer 415 in new implementations, see `aip-137-content-type.md`).
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

#### PUT is a replace, all the way down

The request body is the resource's complete new representation. A field the client
left out is **cleared**, never carried over from the stored value. This holds
recursively: an omitted sub-object resets that whole sub-object, and an omitted array
is the same as an empty one.

Use `Shared.UpsertRequest<T>` so required fields stay required — `Shared.UpdateRequest<T>`
makes every property optional and belongs to PATCH. A `@put` operation paired with
`UpdateRequest<T>` publishes a contract that says "everything is optional" while the
server still rejects a missing required field.

Implementing it takes both layers:

- **v3 handler / `convert.go`** builds the complete target state. No `if body.X != nil { … }`
  guards on a PUT path — always hand the service a value, the zero value included.
- **Adapter** uses the generated `SetOrClear<Field>` helpers (nil clears), never Ent's
  stock `SetNillable<Field>` (nil is a no-op). This choice is the replace-vs-merge switch.
- **`Equal()` short-circuits** must compare an omitted field against the stored value, or
  they skip the write that would have cleared it.
- **`applyTo` / validation projections** must apply the same clearing, so validation sees
  the state the update is about to produce.

Two things stay out of the replace surface:

- **Fields the representation does not expose**, such as server-managed annotations.
  A client cannot send them, so a PUT cannot clear them.
- **Partial internal mutations.** Lifecycle operations that move one attribute (publishing
  or archiving a plan shifts only its effective period) must not ride the general update
  path, or they will clear everything the caller did not repeat. Give them their own named
  adapter method — see `UpdatePlanEffectivePeriod` / `UpdateAddonEffectivePeriod`.

The one deliberate exception is a write-only credential with no cleared state, such as the
Stripe app's `secret_api_key`: make the field **required** so nothing silently persists,
rather than clearing a key the app cannot run without.

PATCH is the partial-update verb and v3 currently ships it only for `update-feature`;
adding it for the remaining entities per the AIP-134 mandate is outstanding.

## AIP-135 delete rules

- DELETE returns `204 No Content` on success.
- **No request body is accepted**; all parameters go in the URL path, not the query string.
- Return `403 Forbidden` before `404 Not Found` — check permissions before existence to prevent resource enumeration.
- Soft-deleted resources return `404 Not Found` on subsequent DELETE calls.
- Cascading deletes:
  - If no protected resources would be affected, proceed as a normal DELETE.
  - If protected resources would be affected and `?force=true` is **not** supplied, return `400 Bad Request` listing the affected resource types in the error detail.
  - If `?force=true` is supplied, delete all child resources and associations, returning `204`.
