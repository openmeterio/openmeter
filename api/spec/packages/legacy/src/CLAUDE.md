# src

<!-- archie:ai-start -->

> Root namespace package for the v1 OpenMeter TypeSpec spec: main.tsp is the sole composition entry point that imports every sibling file and sub-domain folder, declares the @service/@info/@server/@tagMetadata metadata, and exposes shared primitives (types.tsp, errors.tsp, filter.tsp, query.tsp, rest.tsp, auth.tsp) consumed by every sub-domain. Schema-only — compiled to api/openapi.yaml + Go/JS/Python SDKs via make gen-api; no runtime code lives here.

## Patterns

**main.tsp as sole compilation entry point** — main.tsp imports every sibling .tsp and sub-folder; @service/@info/@server/@tagMetadata live ONLY here. A new sub-domain not imported here is silently excluded from the compiled OpenAPI. (`import "./billing";
import "./entitlements/v2";
@service(#{ title: "OpenMeter API" })
namespace OpenMeter;`)
**@friendlyName on every named declaration** — Every model, union, enum, and interface carries @friendlyName for stable Go/JS/Python type names; omitting it yields auto-generated names that break SDK contracts on rename. (`@friendlyName("MeterAggregation")
enum MeterAggregation { SUM, COUNT, ... }`)
**@visibility lifecycle decorators on every field** — Fields declare @visibility(Lifecycle.Read|Create|Update). A field without @visibility is exposed across all lifecycles, leaking internal state on create/update requests. (`@visibility(Lifecycle.Read, Lifecycle.Create)
slug: Key;`)
**Shared primitives centralised in types.tsp / errors.tsp** — ULID, Key, DateTime, Resource, CurrencyCode, Numeric, Annotations (types.tsp) and the RFC 7807 Error model + typed aliases (errors.tsp) are declared once. Never re-declare these in a sub-domain file. (`...OmitProperties<global.Resource, "name">;  // reuse, do not redeclare`)
**Spread / OmitProperties / ResourceCreateModel for composition** — Compose models with ...Other, ...OmitProperties<T,'field'>, or ResourceCreateModel/ResourceReplaceModel<T>. `extends` changes generated property semantics and is reserved for Error subtypes. (`model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;`)
**@sharedRoute for content-negotiated operations** — When one operationId serves multiple content types (JSON vs CSV meter query, single vs batch event ingest), each variant carries @sharedRoute to collapse under one operationId. (`@get @route("/{meterIdOrSlug}/query") @operationId("queryMeter") @sharedRoute
queryJson(...): { @header contentType: "application/json"; ... };`)
**@extension("x-omitempty", true) on all filter operator fields** — Every filter model field ($eq,$in,$like,$and,$or...) in filter.tsp carries @extension("x-omitempty", true) to suppress zero values. New filter operators must follow this. (`@extension("x-omitempty", true)
$eq?: string | null;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Compilation entry point: imports all sibling .tsp + sub-folders; declares @service/@info/@server/@tagMetadata for the entire v1 spec. | Forgetting to import a new sub-domain (it disappears from OpenAPI); duplicating @service or @tagMetadata in a child file. |
| `types.tsp` | Shared primitive scalars/base models: ULID, Key, ExternalKey, DateTime, Resource, ResourceTimestamps, CadencedResource, Annotations, Numeric, Percentage, CurrencyCode, CountryCode. | Re-declaring any primitive in a sub-domain (name collision); a datetime field without @encode(DateTimeKnownEncoding.rfc3339). |
| `errors.tsp` | RFC 7807 Error model (x-go-type models.StatusProblem) + typed error aliases (BadRequestError..GatewayTimeoutError) and CommonErrors / CommonErrorsWithValidation aliases. | Using raw Error in operation unions instead of a typed alias; omitting @error on new error models. |
| `filter.tsp` | Reusable filter models: FilterString, FilterTime, FilterInteger, FilterFloat, FilterBoolean, FilterIDExact. | Adding operators without @extension("x-omitempty", true); creating per-endpoint filter models instead of reusing these. |
| `query.tsp` | Pagination/ordering primitives: QueryPagination, QueryLimitOffset, QueryCursorPagination, PaginatedResponse<T>, CursorPaginatedResponse<T>, QueryOrdering<T>. | Inlining pagination params instead of spreading ...QueryCursorPagination; @body on GET list operations. |
| `rest.tsp` | OpenMeter.Rest.ResourceCreateModel/UpdateModel/ReplaceModel visibility-filtered generics augmenting TypeSpec.Rest.Resource. | Using the TypeSpec built-in ResourceCreateModel directly — it lacks the withVisibilityFilter restricting to Create-visible fields. |
| `events.tsp` | Event ingest (single/batch/JSON @sharedRoute) + v1 limit list and v2 cursor list. Event model carries x-go-type event.Event (cloudevents SDK). | Changing the Event x-go-type extension (maps to a third-party type); @body on v2 list query params; missing @sharedRoute on overloaded ingest ops. |
| `meters.tsp` | Meter CRUD + query interface; MeterCreate/Update via ResourceCreateModel/ResourceReplaceModel; MeterAggregation/WindowSize enums; @sharedRoute JSON vs CSV query. | Omitting @sharedRoute on paired JSON/CSV ops; missing @visibility on new Meter fields; missing @operationId. |

## Anti-Patterns

- Defining new domain models in root-level .tsp files instead of a sub-folder with its own main.tsp
- Re-declaring primitives (ULID, DateTime, Key, Resource) in sub-domain files — they live in types.tsp under namespace OpenMeter
- Using `extends Error` for a new error model without the @error decorator — breaks OpenAPI error schema generation
- Omitting @operationId on interface operations — yields non-deterministic generated SDK function names
- Adding a new sub-domain file without registering it in main.tsp imports — silently excluded from the compiled spec

## Decisions

- **Shared cross-cutting types in types.tsp/errors.tsp, domain types in sub-folders** — Keeps ULID/DateTime/CurrencyCode/error models consistent across all v1 endpoints; sub-folder isolation prevents merge conflicts between billing/entitlements/notification teams.
- **Resource base models (Resource, ResourceTimestamps, CadencedResource) centralised in types.tsp** — Guarantees every resource shares identical id/name/createdAt/updatedAt/deletedAt fields and visibility rules across the whole v1 surface.
- **Distinct QueryPagination, QueryLimitOffset, QueryCursorPagination in query.tsp** — Different endpoints use different pagination strategies; centralising keeps SDK pagination types consistent and avoids per-endpoint parameter drift.

## Example: Add a v1 cursor-paginated list endpoint reusing shared filter + pagination types

```
import "@typespec/http";
using TypeSpec.Http;
using TypeSpec.OpenAPI;
namespace OpenMeter;

@route("/api/v1/widgets")
@tag("Widgets")
@friendlyName("Widgets")
interface WidgetsEndpoints {
  @list
  @operationId("listWidgets")
  @summary("List widgets")
  list(...QueryCursorPagination): CursorPaginatedResponse<Widget> | CommonErrors;
}
```

<!-- archie:ai-end -->
