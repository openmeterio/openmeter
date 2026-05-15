# events

<!-- archie:ai-start -->

> TypeSpec definitions for the metering event ingest and query API: CloudEvents-based MeteringEvent model, IngestedEvent with metadata, filter model, and operations for batch ingest and cursor-paginated listing. All ingest operations are fire-and-forget (202) with three @sharedRoute overloads per content-type.

## Patterns

**@sharedRoute for content-type overloads** — The ingest endpoint has three operation declarations (single CE, batch CE, JSON) all with the same @operationId and @sharedRoute so they compile to one OpenAPI operation with multiple requestBody content types. (`@post @operationId("ingest-metering-events") @sharedRoute ingestEvent(@header contentType: "application/cloudevents+json", @body body: MeteringEvent): IngestEventsResponse | Common.ErrorResponses;`)
**Cursor pagination for event listing** — Events use ...Common.CursorPaginationQuery (not PagePaginationQuery) and return Shared.CursorPaginatedResponse<IngestedEvent>. Filter and sort params are separate named query params. (`@get list(...Common.CursorPaginationQuery, @query filter?: ListEventsParamsFilter, @query sort?: Common.SortQuery): Shared.CursorPaginatedResponse<IngestedEvent> | Common.ErrorResponses;`)
**Filter model uses Common.*FieldFilter types** — ListEventsParamsFilter fields reference Common.StringFieldFilter, Common.ULIDFieldFilter, Common.DateTimeFieldFilter — never inline filter logic or define custom filter operators. (`model ListEventsParamsFilter { id?: Common.StringFieldFilter; customer_id?: Common.ULIDFieldFilter; time?: Common.DateTimeFieldFilter; }`)
**Ingest response is 202 Accepted with empty body** — IngestEventsResponse declares @statusCode _: 202 and no @body, reflecting async processing semantics. Ingest is fire-and-forget, not paginated. (`model IngestEventsResponse { @statusCode _: 202; }`)
**CloudEvents field constraints** — MeteringEvent fields follow CloudEvents spec: id/source/type/subject have @minLength(1); specversion has a default of '1.0'; data is Record<unknown>. (`@minLength(1) @format("uri-reference") source: string;`)
**deepObject filter params with explode** — Filter parameters use @query(#{ style: "deepObject", explode: true }) so filter[field][op]=value query strings deserialize correctly on the server side. (`@query(#{ style: "deepObject", explode: true }) filter?: ListEventsParamsFilter,`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event.tsp` | MeteringEvent (CloudEvents schema) and IngestedEvent (with ingested_at, stored_at, validation_errors). Core model file. | IngestedEventValidationError has code+message+attributes — new validation error codes added in the Go layer must have corresponding documentation here. |
| `operations.tsp` | EventsOperations interface with list (cursor-paginated GET) and three @sharedRoute POST overloads for ingest. Filter uses @query(#{style:"deepObject", explode:true}). | All three ingest overloads share the same @operationId. Adding a fourth content-type overload must also use @sharedRoute and the same operationId to avoid generating a separate OpenAPI operation. |
| `index.tsp` | Barrel import: re-exports event.tsp and operations.tsp so callers only need import ./events/index.tsp. | Every new .tsp file added to this folder must be imported here. |

## Anti-Patterns

- Adding pagination to the ingest endpoint — ingest is fire-and-forget 202, not paginated
- Using PagePaginationQuery for event listing — events use cursor pagination (after/before), not page-number pagination
- Defining filter operators inline in operations instead of using Common.*FieldFilter types
- Removing @sharedRoute from any of the ingest overloads — this collapses three content-type variants to one in the OpenAPI output
- Adding response body fields to IngestEventsResponse — ingest acknowledgment is intentionally empty

## Decisions

- **Three @sharedRoute ingest overloads instead of a single union body** — OpenAPI requires distinct requestBody entries per content-type for correct SDK code generation; TypeSpec @sharedRoute expresses this without duplicating the operation ID.
- **Cursor pagination for events** — Event streams grow continuously and page-number pagination is unstable on append-only data; cursor-based (after/before) pagination gives stable traversal.

## Example: Add a new filter field to event listing

```
// In operations.tsp, add to ListEventsParamsFilter:
model ListEventsParamsFilter {
  // ... existing fields ...

  /**
   * Filter events by event type.
   */
  type?: Common.StringFieldFilter;

  /**
   * Filter events by validation error code.
   */
  validation_error_code?: Common.StringFieldFilterExact;
}
```

<!-- archie:ai-end -->
