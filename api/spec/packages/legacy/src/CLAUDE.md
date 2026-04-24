# src

<!-- archie:ai-start -->

> Root namespace package for the v1 OpenMeter TypeSpec spec; assembles all sub-domain .tsp files through main.tsp imports and declares the service title, version, server URLs, and tag metadata. Nothing is defined here directly — it is the composition root that controls compile order and shared primitive types (types.tsp, errors.tsp, filter.tsp, query.tsp, rest.tsp, auth.tsp) used by every sub-domain.

## Patterns

**main.tsp as the sole compilation entry point** — main.tsp imports every sibling file and sub-folder; the @service, @info, @server, and @tagMetadata decorators live here and nowhere else. New sub-domains must be added as an import here to be included in the generated OpenAPI. (`import "./billing";
import "./productcatalog";`)
**@friendlyName on all model/union/enum/interface declarations** — Every named type must carry @friendlyName to produce stable, human-readable Go/JS/Python type names. Omitting it causes auto-generated names that break SDK contracts on any rename. (`@friendlyName("MeterAggregation")
enum MeterAggregation { SUM, COUNT, ... }`)
**@visibility lifecycle decorators on every model field** — Fields must declare @visibility(Lifecycle.Read), @visibility(Lifecycle.Create), or a combination. Fields without @visibility are exposed across all lifecycles, leaking internal state on create/update. (`@visibility(Lifecycle.Read)
id: ULID;`)
**Spread (...) for model composition instead of inheritance** — Use ...OtherModel or ...OmitProperties<T, 'field'> to compose models. Using `extends` changes property inheritance semantics in the generated OpenAPI output and should be avoided for domain models. (`model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;`)
**Shared primitive types centralised in types.tsp** — ULID, Key, ExternalKey, ULIDOrKey, DateTime, Resource, ResourceTimestamps, CurrencyCode, ISO8601Duration, Numeric, Annotations — all live in types.tsp. Never re-declare these in sub-domain files; import from the root namespace. (`scalar ULID extends string;
scalar DateTime extends utcDateTime;`)
**Filter models with @extension("x-omitempty", true) on every operator field** — All filter model fields ($eq, $in, $like, $and, etc.) in filter.tsp use @extension("x-omitempty", true) to suppress zero values in generated output. New filter operators must follow this pattern. (`@extension("x-omitempty", true)
$eq?: string | null;`)
**@encode(DateTimeKnownEncoding.rfc3339) on DateTime scalar** — The custom DateTime scalar explicitly encodes to RFC 3339 via @encode. Any new datetime scalar must carry the same encoding annotation; omitting it defaults to seconds-since-epoch serialization. (`@encode(DateTimeKnownEncoding.rfc3339)
scalar DateTime extends utcDateTime;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Compilation entry point. Imports all sibling .tsp files and sub-folders; declares @service, @info, @server, @tagMetadata for the entire v1 spec. | Forgetting to add a new sub-domain import here; accidentally duplicating @service or @tagMetadata in child files. |
| `types.tsp` | Shared primitive scalars and base models: ULID, Key, ExternalKey, DateTime, Resource, ResourceTimestamps, CadencedResource, Metadata, Annotations, ISO8601Duration, Numeric, CurrencyCode, CountryCode. | Re-declaring any of these in sub-domain files — causes name collisions in the OpenMeter namespace; adding a new datetime field without @encode(DateTimeKnownEncoding.rfc3339). |
| `errors.tsp` | Defines the canonical RFC 7807 Error model and all typed error aliases (BadRequestError, UnauthorizedError, NotFoundError, etc.) plus CommonErrors and CommonErrorsWithValidation aliases. | Adding a new error status code not present here; using the raw Error model in an operation union instead of the typed alias; omitting @error decorator on new error models. |
| `filter.tsp` | Defines reusable filter models (FilterString, FilterTime, FilterInteger, FilterFloat, FilterBoolean, FilterIDExact) for query parameter filtering across events, meters, and other list endpoints. | Adding new filter operators without @extension("x-omitempty", true); creating per-endpoint filter models instead of reusing these shared types. |
| `query.tsp` | Pagination and ordering primitives: QueryPagination (page/pageSize), QueryLimitOffset, QueryCursorPagination, PaginatedResponse<T>, CursorPaginatedResponse<T>, QueryOrdering<T>, SortOrder. | Defining per-endpoint pagination params inline instead of spreading ...QueryPagination or ...QueryCursorPagination; using @body on GET list operations. |
| `rest.tsp` | Augments TypeSpec.Rest.Resource namespace with ResourceReplaceModel, and defines OpenMeter.Rest.ResourceCreateModel, ResourceUpdateModel, ResourceReplaceModel visibility-filtered generics. | Using the TypeSpec built-in ResourceCreateModel directly without going through OpenMeter.Rest.ResourceCreateModel — the built-in lacks the withVisibilityFilter ensuring only Create-visible fields are included. |
| `meters.tsp` | Meter CRUD + query interface (MetersEndpoints), MeterCreate/Update/Query models, MeterAggregation/WindowSize enums. Uses @sharedRoute for content-negotiated JSON vs CSV query responses. | Adding non-idempotent query params without @query decorator; omitting @sharedRoute on content-negotiated paired operations; forgetting @visibility annotations on new Meter fields. |
| `events.tsp` | Event ingestion (ingestEvent/ingestEvents with @sharedRoute per content-type) and listing (listEvents v1 + listEventsV2 cursor-paginated). Defines Event (maps to sdk-go event.Event via x-go-type), IngestedEvent, IngestEventsBody union. | Changing the Event model's x-go-type extension — it maps to a third-party type; adding @body to v2 list query params; missing @sharedRoute on overloaded ingest operations. |

## Anti-Patterns

- Defining new domain models directly in root-level .tsp files (auth, debug, errors, filter, main, meters, portal, query, rest, subjects, types) instead of creating a sub-folder with its own main.tsp
- Re-declaring primitive types (ULID, DateTime, Key, Resource, etc.) in sub-domain files — they already live in types.tsp under the OpenMeter namespace
- Using `extends Error` for new error models without the @error decorator — breaks OpenAPI error schema generation
- Omitting @operationId on interface operations — causes non-deterministic generated function names in SDKs
- Adding new sub-domain files without registering them in main.tsp imports — they will be silently excluded from the compiled spec

## Decisions

- **Shared primitive types in types.tsp and errors.tsp, sub-domain types in sub-folders** — Keeps cross-cutting types (ULID, DateTime, CurrencyCode, error models) consistent across all v1 endpoints without duplication; sub-folder isolation prevents merge conflicts between teams working on billing vs entitlements vs notifications.
- **Resource base models (Resource, ResourceTimestamps, CadencedResource) in types.tsp rather than per-domain** — Ensures all resources share identical id/name/createdAt/updatedAt/deletedAt fields and visibility rules, enforcing API consistency across the entire v1 surface.
- **Separate QueryPagination (page/pageSize), QueryLimitOffset, and QueryCursorPagination models in query.tsp** — Different endpoints use different pagination strategies; centralising them ensures SDK-level pagination types remain consistent and avoids per-endpoint drift in parameter names or defaults.

## Example: Adding a new list endpoint with cursor pagination and a filter body param

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
