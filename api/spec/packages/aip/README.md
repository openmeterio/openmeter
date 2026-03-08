# AIP (`packages/aip`)

This package defines OpenMeter v3 and Konnect metering & billing APIs using TypeSpec, following [Kong's AIP (API Improvement Proposals)](https://kong-aip.netlify.app/list/).

**Outputs:** `openapi.MeteringAndBilling.yaml` (OpenMeter + Konnect variants)

---

## Naming conventions ([AIP-122](https://kong-aip.netlify.app/aips/122))

| Element            | Convention   | Example                                 |
| ------------------ | ------------ | --------------------------------------- |
| Model names        | `PascalCase` | `BillingProfile`, `MeterQueryResult`    |
| Model properties   | `snake_case` | `created_at`, `billing_profile_id`      |
| Path parameters    | `camelCase`  | `meterId`, `customerId`                 |
| Enum names         | `PascalCase` | `MeterAggregation`                      |
| Enum member names  | `PascalCase` | `UniqueCount`                           |
| Enum values (wire) | `snake_case` | `"unique_count"`                        |
| Operation IDs      | `kebab-case` | `create-meter`, `list-billing-profiles` |

These are enforced by the linter rules in `lib/rules/` and produce errors or warnings at compile time.

---

## Enums ([AIP-126](https://kong-aip.netlify.app/aips/126))

- All enum wire values must be `snake_case` (enforced as an error by the `casing-aip-errors` rule).
- Every enum must define an `Unknown` member as the zero/default value.
- Prefer enums over booleans for two-state fields — this allows a third state to be added later without a breaking change.

---

## Base resource models ([AIP-122](https://kong-aip.netlify.app/aips/122))

All resources spread `Shared.Resource` (or `Shared.ResourceWithKey` for resources with a user-defined key). These provide the standard fields with correct `@visibility` already applied.

**`Shared.Resource`** — `id`, `name`, `description`, `labels`, `created_at`, `updated_at`, `deleted_at`

**`Shared.ResourceWithKey`** — same as above plus `key: ResourceKey` (`Lifecycle.Read, Lifecycle.Create` only)

```tsp
model Meter {
  ...Shared.Resource;

  @visibility(Lifecycle.Read, Lifecycle.Create)
  event_type: string;
}
```

---

## Visibility

Use `@visibility` on every field to control which operations expose it. Do not rely on defaults.

- `Lifecycle.Read` — returned by any operation (GET, list, create/update response, etc.)
- `Lifecycle.Create` — accepted in create request bodies (POST)
- `Lifecycle.Update` — accepted in update request bodies (PATCH)

Server-managed fields (`id`, timestamps) must be `Lifecycle.Read` only.

---

## Request/response templates ([AIP-134](https://kong-aip.netlify.app/aips/134), [AIP-135](https://kong-aip.netlify.app/aips/135))

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

**AIP-134 update rules:**

- `PATCH` implements partial update (JSON Merge Patch, RFC 7396). Operation ID: `update-<entity>`.
- `PUT` is a create-or-replace upsert. Returns `201` on creation and `200` on replacement. Operation ID: `upsert-<entity>` or `update-<entity>`.
- `PATCH` must reject unknown fields and read-only fields with `400 Bad Request`.

**AIP-135 delete rules:**

- DELETE returns `204 No Content`. No request body is accepted.
- Return `403 Forbidden` before `404 Not Found` — check permissions before existence to prevent resource enumeration.

---

## List endpoints ([AIP-132](https://kong-aip.netlify.app/aips/132))

List endpoints are `GET` requests to the entity root (e.g. `GET /meters`). Items go under the `data` key provided by the pagination response templates.

**Sort order is mandatory** — it must be deterministic. Prefer `name asc`, `created_at asc`, or `updated_at desc`. Never use a UUID as the sort key.

Expose sorting via `@query(#{ name: "sort" }) sort?: Common.SortQuery`. Sort values are comma-delimited attribute names with optional `asc`/`desc` suffixes (e.g. `name,created_at desc`). Sortable attributes should include all filterable attributes. Return `400` if a sort value cannot be executed.

---

## Pagination ([AIP-158](https://kong-aip.netlify.app/aips/158))

Pagination must be supported from day one — adding it later is a breaking change.

**Page-based** (for low-cardinality or offset-friendly collections):
Spread `...Common.PagePaginationQuery` into the operation. Parameters: `page[size]`, `page[number]`.
Return `Shared.PagePaginatedResponse<T>` — response includes `meta.page.{number, size, total}`.

**Cursor-based** (preferred for large or frequently-updated collections):
Spread `...Common.CursorPaginationQuery` into the operation. Parameters: `page[size]`, `page[after]`, `page[before]` (`after` and `before` are mutually exclusive — using both returns `400`).
Return `Shared.CursorPaginatedResponse<T>` — response includes `meta.page.{first, last, next, previous, size}`.

Cursors must be opaque to clients. `next`/`previous` URIs in the response must preserve all original query parameters except pagination ones.

---

## Filtering ([AIP-160](https://kong-aip.netlify.app/aips/160))

Filters use `?filter[field]=value` query syntax. Expose them as a `filter` parameter with `style: "deepObject", explode: true`. Return `400 Bad Request` for unsupported filter fields.

Define a named filter model for each list operation. Spread `Shared.ResourceFilters` to get the standard resource fields (`name`, `labels`, `public_labels`, `created_at`, `updated_at`, `deleted_at`) for free.

Use the filter types from `common/parameters.tsp` (backed by the shared `aip_filters.yaml`):

| Type                             | Use for                         | Example query param                            |
| -------------------------------- | ------------------------------- | ---------------------------------------------- |
| `Common.StringFieldFilter`       | string, partial or exact match  | `filter[name][contains]=foo`                   |
| `Common.StringFieldFilterExact`  | string, exact match only        | `filter[key]=my-key`                           |
| `Common.BooleanFieldFilter`      | boolean                         | `filter[active]=true`                          |
| `Common.NumericFieldFilter`      | numeric comparisons             | `filter[amount][gte]=10`                       |
| `Common.DateTimeFieldFilter`     | RFC-3339 datetime comparisons   | `filter[created_at][gte]=2024-01-01T00:00:00Z` |
| `Common.LabelsFieldFilter`       | labels map, dot-notation key    | `filter[labels.env]=prod`                      |
| `Common.PublicLabelsFieldFilter` | public_labels map, dot-notation | `filter[public_labels.tier]=free`              |

**Operators** (appended as `[op]`): `eq` (default), `neq`, `oeq` (OR-equal, comma-separated), `contains`, `ocontains` (OR-contains), `lt`, `lte`, `gt`, `gte`. `filter[field]` without a value matches records where the field is not null.

**Label filtering** uses dot-notation to address individual keys: `filter[labels.owner]=alice`, `filter[labels.env][ocontains]=dev,test`.

> **Note:** `Shared.QueryFilter*` in `shared/filters.tsp` are structured filter objects for JSON request bodies (e.g. POST query endpoints). Use `Common.*FieldFilter` for URL query parameters on list endpoints.

---

## Labels ([AIP-129](https://kong-aip.netlify.app/aips/129))

`Common.Labels` stores mutable user-managed metadata. `Common.PublicLabels` is for publicly visible labels. Both are included in `Shared.Resource` via the `labels` and `public_labels` fields.

Key constraints: 1–63 characters, cannot start with `kong`, `konnect`, `mesh`, `kic`, or `_`.

Labels support filtering via dot-notation (see Filtering section above).

---

## Error responses ([AIP-193](https://kong-aip.netlify.app/aips/193))

All errors follow RFC 7807 (`Content-Type: application/problem+json`) with mandatory fields: `type`, `status`, `title`, `detail`, `instance` (correlation ID). All `400` responses include an `invalid_parameters` array with `field`, `reason`, `source`, and `rule`.

Use `Common.ErrorResponses` (= `BadRequest | Unauthorized | Forbidden`) on every operation, then add specific types explicitly:

| Type                          | Status | When to add explicitly                       |
| ----------------------------- | ------ | -------------------------------------------- |
| `Common.NotFound`             | 404    | GET, PATCH, PUT, DELETE by ID                |
| `Common.Gone`                 | 410    | PUT/PATCH when the resource was soft-deleted |
| `Common.Conflict`             | 409    | Create operations that may conflict          |
| `Common.UnprocessableContent` | 422    | Semantically invalid requests                |

```tsp
get(@path meterId: Shared.ULID): Shared.GetResponse<Meter> | Common.NotFound | Common.ErrorResponses;
```

Access control rule: return `403` for resources the caller owns but lacks permission for; `404` for everything else — this prevents resource enumeration.

---

## Composition over inheritance

The `composition-over-inheritance` linter rule (warning) flags `extends` on a base without `@discriminator`. Prefer:

- **Spread** (`...BaseModel`) for composing fields into a new model.
- **`model Foo is Bar`** for aliasing or narrowing a template instantiation.
- **`@discriminator`** on the base model only when a true polymorphic union is needed.

---

## Documentation requirements

The `doc-decorator` linter rule (warning) requires a `/** comment */` or `@doc` on:

- All named models, enums, and unions.
- All model properties, except `_` and `contentType`.

Operations must have both `@operationId` (kebab-case) and `@summary`.

Use `#suppress "@openmeter/api-spec-aip/doc-decorator" "<reason>"` only for shared base model spreads where documenting each inherited property individually would be redundant.

---

## Stability markers ([AIP-181](https://kong-aip.netlify.app/aips/181))

Mark non-public operations with `@extension` using the constants from `shared/consts.tsp`:

| Constant                   | Extension key | Meaning                                   |
| -------------------------- | ------------- | ----------------------------------------- |
| `Shared.PrivateExtension`  | `x-private`   | Not exposed in public documentation       |
| `Shared.UnstableExtension` | `x-unstable`  | Subject to breaking changes               |
| `Shared.InternalExtension` | `x-internal`  | Hidden from the API documentation website |
