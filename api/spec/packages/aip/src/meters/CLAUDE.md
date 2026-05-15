# meters

<!-- archie:ai-start -->

> TypeSpec definitions for the Meters v3 API domain: Meter model (aggregation config, event_type, JSONPath dimensions), query request/result types, and CRUD+query operations. Primary constraint: query types live in query.tsp (not meter.tsp) for cross-namespace reuse.

## Patterns

**union over enum for open enumerations** — MeterAggregation and MeterQueryGranularity use `union` with string literals — not `enum` — so generated SDKs treat them as open string unions, preserving forward-compatibility when new aggregation types are added. (`union MeterAggregation { sum: "sum", count: "count", unique_count: "unique_count", avg: "avg", min: "min", max: "max", latest: "latest" }`)
**@sharedRoute for content-negotiated variants** — The meter query endpoint has two operations (JSON and CSV) sharing the same POST URL via @sharedRoute. Content type is discriminated by @header contentType in the response. The CSV variant (queryCsv) intentionally omits @body on the request to avoid anyOf in generated OpenAPI. (`@post @operationId("query-meter") @sharedRoute query(@path meterId: Shared.ULID, @body request: MeterQueryRequest): { @header contentType: "application/json"; @body _: MeterQueryResult; } | ...`)
**@example decorator on complex models** — Models with complex structure (Meter, MeterQueryRequest, MeterQueryRow, MeterQueryResult) include @example(#{ ... }) decorators to provide concrete OpenAPI examples in the generated spec. (`@example(#{ id: "01G65Z755AFWAKHE12NY0CQ9FH", key: "tokens_total", aggregation: "sum", event_type: "prompt", value_property: "$.tokens" }) model Meter { ... }`)
**query.tsp isolation for reusable query types** — MeterQueryRequest, MeterQueryResult, MeterQueryRow, MeterQueryFilters, and MeterQueryGranularity are isolated in query.tsp so they can be imported by other namespaces (e.g. features/cost.tsp imports query.tsp to reuse MeterQueryRequest). (`// query.tsp imported by features/cost.tsp:
import "../meters/query.tsp";
// then uses Meters.MeterQueryRequest as cost query body`)
**@visibility for immutable vs. updatable fields** — Meter fields aggregation, event_type, and value_property carry only Lifecycle.Read + Lifecycle.Create (immutable after creation). The dimensions field uniquely adds Lifecycle.Update — it is the only mutable Meter field. (`@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) dimensions?: Record<string>; // vs. @visibility(Lifecycle.Read, Lifecycle.Create) aggregation: MeterAggregation;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `operations.tsp` | Declares MetersOperations (CRUD) and MetersQueryOperations (query + queryCsv). Both query methods share operationId 'query-meter' via @sharedRoute. | queryCsv intentionally omits @body on the request. Do not add a request body back — it would produce anyOf in the generated OpenAPI spec. |
| `meter.tsp` | Core Meter model with aggregation type, event_type, optional value_property, and dimensions map. | aggregation and event_type are Create+Read only (immutable after creation). value_property is optional only for count aggregation. |
| `query.tsp` | All meter query I/O types. MeterQueryFilters.dimensions uses Record<Shared.QueryFilterStringMapItem> for dimension filtering. MeterQueryRow.dimensions is Record<string> (output only). | group_by_dimensions has @maxItems(100). Do not confuse input filter map (QueryFilterStringMapItem) with output dimensions map (plain Record<string>). |
| `index.tsp` | Barrel — imports meter.tsp then operations.tsp. Note: query.tsp is NOT imported here; it is imported directly by operations.tsp and by external namespaces that need only query types. | If adding a new .tsp file to this folder, import it in index.tsp or it will be invisible to consumers using the barrel import. |

## Anti-Patterns

- Using `enum` instead of `union` for MeterAggregation or MeterQueryGranularity — breaks forward-compatibility of generated SDKs
- Adding a request body to the queryCsv operation — intentionally omitted to prevent anyOf in OpenAPI output
- Putting query types (MeterQueryRequest, MeterQueryResult, etc.) in meter.tsp — they belong in query.tsp for cross-namespace reuse by features/cost.tsp
- Setting dimensions field visibility to Read+Create only — dimensions is the one updatable Meter field and must keep Lifecycle.Update

## Decisions

- **CSV export as @sharedRoute variant of the same POST endpoint rather than a separate GET endpoint** — Content negotiation (Accept: text/csv) on a single URL is idiomatic REST; sharedRoute avoids duplicating route+operationId while allowing separate TypeSpec response type shapes for JSON vs. CSV.
- **Query types isolated in query.tsp separate from meter.tsp** — features/cost.tsp needs to import Meters.MeterQueryRequest without pulling in the full Meter model and its CRUD operations; isolation enables targeted cross-namespace imports.

## Example: Adding a new meter operation (e.g. bulk-delete) following existing patterns

```
// operations.tsp — inside interface MetersOperations:
@post
@operationId("bulk-delete-meters")
@summary("Bulk delete meters")
@route("/bulk-delete")
@extension(Shared.UnstableExtension, true)
bulkDelete(
  @body request: BulkDeleteMetersRequest,
): Shared.DeleteResponse | Common.ErrorResponses;

// meter.tsp:
@friendlyName("BulkDeleteMetersRequest")
model BulkDeleteMetersRequest {
  @minItems(1) @maxItems(100)
  ids: Shared.ULID[];
// ...
```

<!-- archie:ai-end -->
