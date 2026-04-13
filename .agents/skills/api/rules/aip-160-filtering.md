# AIP-160 — Filtering

Reference: https://kong-aip.netlify.app/aip/160/

Filters use `?filter[field]=value` query syntax. Expose them as a `filter` parameter with `style: "deepObject", explode: true`. Return `400 Bad Request` for unsupported filter fields, with the unsupported field listed in `invalid_parameters`.

Define a named filter model for each list operation. Spread `Shared.ResourceFilters` (`api/spec/packages/aip/src/shared/parameters.tsp`) to get these standard resource filter fields for free:

- `name` — `Common.StringFieldFilter`
- `labels` — `Common.LabelsFieldFilter`
- `public_labels` — `Common.LabelsFieldFilter` (note: uses the regular `LabelsFieldFilter`, **not** `PublicLabelsFieldFilter`)
- `created_at` — `Common.DateTimeFieldFilter`
- `updated_at` — `Common.DateTimeFieldFilter`
- `deleted_at` — `Common.DateTimeFieldFilter`

Only spread `Shared.ResourceFilters` on resources that actually carry all of these fields (e.g. spreading `Shared.Resource`). For `Shared.ResourceImmutable` resources, add individual filter fields by hand — `updated_at` and `deleted_at` do not exist on the model.

## Filter types

Use the filter types from `api/spec/packages/aip/common/parameters.tsp` (backed by the shared `aip_filters.yaml`). Each maps 1:1 to a Go `filters.Filter*` struct in `api/v3/filters/filter.go`.

| TypeSpec type                    | Use for                              | Example query param                            | Go counterpart               |
| -------------------------------- | ------------------------------------ | ---------------------------------------------- | ---------------------------- |
| `Common.StringFieldFilter`       | string, partial or exact match       | `filter[name][contains]=foo`                   | `filters.FilterString`       |
| `Common.StringFieldFilterExact`  | string, exact match only             | `filter[key]=my-key`                           | `filters.FilterStringExact`  |
| `Common.UuidFieldFilter`         | UUID/ID columns, exact match only    | `filter[id]=3bbfd3a-...`                       | `filters.FilterStringExact`  |
| `Common.BooleanFieldFilter`      | boolean (bare `true`/`false` scalar) | `filter[active]=true`                          | `filters.FilterBoolean`      |
| `Common.NumericFieldFilter`      | numeric comparisons                  | `filter[amount][gte]=10`                       | `filters.FilterNumeric`      |
| `Common.DateTimeFieldFilter`     | RFC-3339 datetime comparisons        | `filter[created_at][gte]=2024-01-01T00:00:00Z` | `filters.FilterDateTime`     |
| `Common.LabelsFieldFilter`       | `labels` map, dot-notation key       | `filter[labels.env]=prod`                      | `filters.FilterString` (dot) |
| `Common.PublicLabelsFieldFilter` | `public_labels` map, dot-notation    | `filter[public_labels.tier]=free`              | `filters.FilterString` (dot) |
| `Common.AttributesFieldFilter`   | `attributes` map, dot-notation       | `filter[attributes.env]=dev`                   | `filters.FilterString` (dot) |

### Operators per OAS type

The authoritative operator surface for each Common type is defined in `api/spec/packages/aip/common/definitions/aip_filters.yaml`. Notably:

- `StringFieldFilter`: implicit-eq, `eq`, `neq`, `contains`, `ocontains`, `oeq` — **no ranges**
- `StringFieldFilterExact` / `UuidFieldFilter`: implicit-eq, `eq`, `neq`, `oeq`
- `NumericFieldFilter`: implicit-eq, `eq`, `lt`, `lte`, `gt`, `gte` — **no `neq` or `oeq`** in the OAS
- `DateTimeFieldFilter`: implicit-eq, `eq`, `lt`, `lte`, `gt`, `gte` — **no `neq`**
- `BooleanFieldFilter`: bare scalar only (`true` / `false`)
- `LabelsFieldFilter` / `PublicLabelsFieldFilter` / `AttributesFieldFilter`: spread of `StringFieldFilter`, addressed via first-dot delimiter

### Go-side operator superset (implementation-only)

The Go types in `api/v3/filters/filter.go` deliberately expose a **superset** of operators beyond what the matching OAS `Common.*FieldFilter` advertises, so a single Go struct can back several endpoints with different narrow OAS surfaces:

- `filters.FilterString` adds `gt`/`gte`/`lt`/`lte` and an `$exists` check on top of `Common.StringFieldFilter`'s operators
- `filters.FilterNumeric` adds `neq` and `oeq`
- `filters.FilterDateTime` matches its Common type exactly (`eq`/`gt`/`gte`/`lt`/`lte`)

The existence check (`$exists`) and `nexists` are AIP-160 operators, but they are not currently modeled in any `Common.*FieldFilter` `oneOf` shape, so picking e.g. `Common.StringFieldFilter` in TypeSpec does not advertise them. Do not rely on `gt`/`gte`/`lt`/`lte` on string fields as a stable contract — the Common type is the public surface.

## Operators

AIP-160 defines these operators, appended as `[op]`:

- `eq` (default when the operator is omitted)
- `neq`
- `oeq` (OR-equal, comma-separated values)
- `contains`
- `ocontains` (OR-contains, comma-separated values)
- `lt`, `lte`, `gt`, `gte` (numeric / datetime ranges)
- **existence check** — a bare parameter `?filter[field]` with no value asserts the field is not null
- `nexists` — asserts the map key is absent; **limited to unschematized map fields** like `labels`, `public_labels`, `attributes`, `metadata`

**Which of these are valid on a given field depends on the Common type chosen above — see the per-OAS-type table.** `aip_filters.yaml` currently only models the equality/containment/range operators in its `oneOf` shapes; `exists` and `nexists` are AIP-160-documented but handled at the query-parameter layer rather than the schema layer.

Boolean and null matching uses the string literals `true`, `false`, `null`, and only applies to `eq`/`neq`.

## Combining filters

- Multiple filters across **different fields** combine with **AND**.
- Comma-separated values on a single field (`oeq` / `ocontains`) combine with **OR**.
- Range queries combine a lower bound (`gt` or `gte`) with an upper bound (`lt` or `lte`) — these are AND-ed by the converter.

## List field quantifiers

`?filter[field][operator][any]=value` or `[all]`. **Not currently supported by the Go parser in `api/v3/filters/`** — document the intent but do not expose quantifier operators on list fields yet.

## Label / map filtering

Dot-notation addresses individual keys on map-typed fields:

- `filter[labels.owner]=alice`, `filter[labels.env][ocontains]=dev,test`
- `filter[public_labels.tier]=free`
- `filter[attributes.env]=dev`

Only the **first** `.` is treated as a delimiter (`.` is itself a legal label-key character), so `filter[labels.a.b.c]=x` means "the label whose key is `a.b.c`".

All three of `Common.LabelsFieldFilter`, `Common.PublicLabelsFieldFilter`, and `Common.AttributesFieldFilter` spread `StringFieldFilter`, so they inherit the same operator surface (`eq`, `neq`, `contains`, `ocontains`, `oeq`).

## Structured body filters vs query filters

`Shared.QueryFilter*` in `shared/filters.tsp` are structured filter objects for JSON **request bodies** (e.g., POST query endpoints). Use `Common.*FieldFilter` for URL query parameters on list endpoints.

## Go-side implementation

For the handler decoder (`filters.Parse`), the adapter (`Convert*` → Ent `.Select(field)`), validation rules, security caps, and gotchas, use the `/api-filters` skill. That skill owns the Go-side contract; this file is the TypeSpec-side contract only.
