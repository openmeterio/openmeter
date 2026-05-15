# src

<!-- archie:ai-start -->

> Root namespace package for the v1 OpenMeter TypeSpec spec; serves as the composition root that assembles all sub-domain .tsp files through main.tsp imports and declares the service title, version, server URLs, and tag metadata. Shared primitive types (types.tsp, errors.tsp, filter.tsp, query.tsp, rest.tsp, auth.tsp) live here and are consumed by every sub-domain.

## Patterns

**main.tsp as sole compilation entry point** — main.tsp imports every sibling file and sub-folder; @service, @info, @server, and @tagMetadata decorators live only here. New sub-domains must be added as an import here to appear in the generated OpenAPI. (`import "./billing";
import "./productcatalog";`)
**@friendlyName on all named type declarations** — Every model, union, enum, and interface must carry @friendlyName to produce stable, human-readable Go/JS/Python type names. Omitting it causes auto-generated names that break SDK contracts on any rename. (`@friendlyName("MeterAggregation")
enum MeterAggregation { SUM, COUNT, ... }`)
**@visibility lifecycle decorators on every model field** — Fields must declare @visibility(Lifecycle.Read), @visibility(Lifecycle.Create), or a combination. Fields without @visibility are exposed across all lifecycles, leaking internal state on create/update requests. (`@visibility(Lifecycle.Read)
id: ULID;`)
**Shared primitive types centralised in types.tsp** — ULID, Key, ExternalKey, DateTime, Resource, ResourceTimestamps, CurrencyCode, ISO8601Duration, Numeric, Annotations all live in types.tsp under OpenMeter namespace. Never re-declare these in sub-domain files. (`scalar ULID extends string;
@encode(DateTimeKnownEncoding.rfc3339)
scalar DateTime extends utcDateTime;`)
**Spread (...) for model composition instead of inheritance** — Use ...OtherModel or ...OmitProperties<T, 'field'> to compose models. Using extends changes property inheritance semantics in generated OpenAPI and should be avoided for domain models. (`model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;`)
**@sharedRoute for content-negotiated operations** — When the same operationId handles multiple content types (e.g., JSON vs CSV meter query, or single vs batch event ingest), each operation variant must carry @sharedRoute to collapse them under one operationId in the OpenAPI spec. (`@get @route("/{meterIdOrSlug}/query") @operationId("queryMeter") @sharedRoute
queryJson(...): { @header contentType: "application/json"; ... };`)
**@extension("x-omitempty", true) on all filter model operator fields** — All filter model fields ($eq, $in, $like, $and, etc.) in filter.tsp use @extension("x-omitempty", true) to suppress zero values in generated output. New filter operators must follow this pattern. (`@extension("x-omitempty", true)
$eq?: string | null;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Compilation entry point. Imports all sibling .tsp files and sub-folders; declares @service, @info, @server, @tagMetadata for the entire v1 spec. | Forgetting to add a new sub-domain import here; accidentally duplicating @service or @tagMetadata in child files. |
| `types.tsp` | Shared primitive scalars and base models: ULID, Key, ExternalKey, DateTime, Resource, ResourceTimestamps, CadencedResource, Metadata, Annotations, ISO8601Duration, Numeric, CurrencyCode, CountryCode. | Re-declaring any of these in sub-domain files causing name collisions; adding a datetime field without @encode(DateTimeKnownEncoding.rfc3339). |
| `errors.tsp` | Defines the canonical RFC 7807 Error model and all typed error aliases (BadRequestError, UnauthorizedError, NotFoundError, ValidationErrorResponse, etc.) plus CommonErrors and CommonErrorsWithValidation aliases. | Using raw Error model in operation unions instead of typed aliases; omitting @error decorator on new error models; adding status codes not present here. |
| `filter.tsp` | Reusable filter models (FilterString, FilterTime, FilterInteger, FilterFloat, FilterBoolean, FilterIDExact) for query parameter filtering. | Adding new filter operators without @extension("x-omitempty", true); creating per-endpoint filter models instead of reusing shared types. |
| `query.tsp` | Pagination and ordering primitives: QueryPagination, QueryLimitOffset, QueryCursorPagination, PaginatedResponse<T>, CursorPaginatedResponse<T>, QueryOrdering<T>, SortOrder. | Defining per-endpoint pagination params inline instead of spreading ...QueryPagination or ...QueryCursorPagination; using @body on GET list operations. |
| `rest.tsp` | Augments TypeSpec.Rest.Resource with ResourceReplaceModel and defines OpenMeter.Rest.ResourceCreateModel, ResourceUpdateModel, ResourceReplaceModel visibility-filtered generics. | Using TypeSpec built-in ResourceCreateModel directly — the built-in lacks withVisibilityFilter ensuring only Create-visible fields are included. |
| `meters.tsp` | Meter CRUD + query interface (MetersEndpoints), MeterCreate/Update models, MeterAggregation/WindowSize enums. Uses @sharedRoute for content-negotiated JSON vs CSV query responses. | Omitting @sharedRoute on content-negotiated paired operations; forgetting @visibility annotations on new Meter fields; omitting @operationId. |
| `events.tsp` | Event ingestion (single, batch, JSON) and listing (v1 limit-based, v2 cursor-paginated). Event model carries x-go-type extension mapping to sdk-go event.Event. | Changing Event model's x-go-type extension (maps to third-party type); adding @body to v2 list query params; missing @sharedRoute on overloaded ingest operations. |

## Anti-Patterns

- Defining new domain models directly in root-level .tsp files instead of creating a sub-folder with its own main.tsp
- Re-declaring primitive types (ULID, DateTime, Key, Resource) in sub-domain files — they already live in types.tsp under the OpenMeter namespace
- Using `extends Error` for new error models without the @error decorator — breaks OpenAPI error schema generation
- Omitting @operationId on interface operations — causes non-deterministic generated function names in SDKs
- Adding new sub-domain files without registering them in main.tsp imports — they are silently excluded from the compiled spec

## Decisions

- **Shared primitive types in types.tsp and errors.tsp; sub-domain types in sub-folders** — Keeps cross-cutting types (ULID, DateTime, CurrencyCode, error models) consistent across all v1 endpoints without duplication; sub-folder isolation prevents merge conflicts between teams working on billing vs entitlements vs notifications.
- **Resource base models (Resource, ResourceTimestamps, CadencedResource) in types.tsp rather than per-domain** — Ensures all resources share identical id/name/createdAt/updatedAt/deletedAt fields and visibility rules, enforcing API consistency across the entire v1 surface.
- **Separate QueryPagination, QueryLimitOffset, and QueryCursorPagination models in query.tsp** — Different endpoints use different pagination strategies; centralising them ensures SDK-level pagination types remain consistent and avoids per-endpoint drift in parameter names or defaults.

## Example: Adding a new list endpoint with cursor pagination and shared filter types

```
import "@typespec/http";
import "@typespec/openapi3";
using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/widgets")
@tag("Widgets")
@friendlyName("Widgets")
interface WidgetsEndpoints {
  @get
  @operationId("listWidgets")
  @summary("List widgets")
  list(
// ...
```

<!-- archie:ai-end -->
