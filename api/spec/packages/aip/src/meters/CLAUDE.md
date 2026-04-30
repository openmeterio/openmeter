# meters

<!-- archie:ai-start -->

> TypeSpec definitions for the Meters domain: Meter model (aggregation config, event_type, dimensions), query request/result types, and CRUD+query operations for the v3 API.

## Patterns

**union for open-ish enumerations** — MeterAggregation and MeterQueryGranularity use `union` (not `enum`) with string literals so generated SDKs treat them as string unions rather than closed enums, supporting forward-compatibility. (`union MeterAggregation { sum: "sum", count: "count", unique_count: "unique_count", avg: "avg", min: "min", max: "max", latest: "latest" }`)
**sharedRoute for content-negotiated variants** — The meter query endpoint has two operations (JSON and CSV) using `@sharedRoute` so both map to the same URL; content type is discriminated by `@header contentType` in the response. (`@post @operationId("query-meter") @sharedRoute
query(@path meterId: Shared.ULID, @body request: MeterQueryRequest): { @header contentType: "application/json"; @body _: MeterQueryResult; } | ...`)
**@example for model documentation** — Models with complex structure (Meter, MeterQueryRequest, MeterQueryRow, MeterQueryResult) include `@example(#{ ... })` decorators to provide concrete OpenAPI examples in the generated spec. (`@example(#{ id: "01G65Z755AFWAKHE12NY0CQ9FH", key: "tokens_total", aggregation: "sum", ... })
model Meter { ... }`)
**Separate query.tsp for read-only query types** — MeterQueryRequest, MeterQueryResult, MeterQueryRow, MeterQueryFilters, and MeterQueryGranularity are isolated in query.tsp (not meter.tsp) so they can be imported by other namespaces (e.g. Features.FeatureCostOperations). (`// features/cost.tsp
import "../meters/query.tsp";
// uses Meters.MeterQueryRequest as request body for queryCost`)

## Key Files

| File             | Role                                                                                                                                                | Watch For                                                                                                                                                         |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `operations.tsp` | Declares MetersOperations (CRUD) and MetersQueryOperations (query+queryCsv). query and queryCsv share the same operationId via @sharedRoute.        | queryCsv intentionally omits @body on the request to avoid anyOf in generated OpenAPI — do not add request body back.                                             |
| `meter.tsp`      | Core Meter model. dimensions field is `@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)` — it is updatable unlike most meter fields. | aggregation and event_type are Create+Read only (immutable after creation). value_property is optional for count aggregation.                                     |
| `query.tsp`      | All meter query I/O types. MeterQueryFilters.dimensions uses Record<Shared.QueryFilterStringMapItem> for dimension filtering.                       | group_by_dimensions has @maxItems(100). dimensions in MeterQueryRow is Record<string> (output), while MeterQueryFilters.dimensions uses filter map items (input). |

## Anti-Patterns

- Using `enum` instead of `union` for MeterAggregation — breaks forward-compatibility of generated SDKs
- Adding request body to queryCsv — intentionally omitted to prevent anyOf in OpenAPI output
- Putting query types (MeterQueryRequest etc.) in meter.tsp — they belong in query.tsp for cross-namespace reuse

## Decisions

- **CSV export handled as a sharedRoute variant of the same POST endpoint rather than a separate GET endpoint** — Content negotiation (Accept: text/csv) on a single URL is idiomatic REST; sharedRoute avoids duplicating route+operationId while allowing separate TypeSpec response type shapes.

<!-- archie:ai-end -->
