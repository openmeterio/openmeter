# events

<!-- archie:ai-start -->

> v3 (AIP) TypeSpec for the metering event ingest and query API: CloudEvents-based MeteringEvent/IngestedEvent models, a filter model, cursor-paginated listing, and batch ingest. Ingest is fire-and-forget (202) with three @sharedRoute overloads per content-type.

## Patterns

**@sharedRoute content-type overloads** — The ingest endpoint has three operation declarations (single CloudEvent, batch CloudEvent, JSON) all with the same @operationId and @sharedRoute, compiling to one OpenAPI operation with multiple requestBody content types. (`@post @operationId("ingest-metering-events") @sharedRoute ingestEvent(@header contentType: "application/cloudevents+json", @body body: MeteringEvent): IngestEventsResponse | Common.ErrorResponses;`)
**Cursor pagination for event listing** — list uses ...Common.CursorPaginationQuery (not PagePaginationQuery) and returns Shared.CursorPaginatedResponse<IngestedEvent>; filter and sort are separate named query params. (`list(...Common.CursorPaginationQuery, @query filter?: ListEventsParamsFilter, @query sort?: Common.SortQuery): Shared.CursorPaginatedResponse<IngestedEvent> | Common.ErrorResponses;`)
**Filter model uses Common.*FieldFilter types** — ListEventsParamsFilter fields reference Common.StringFieldFilter, Common.ULIDFieldFilter, Common.DateTimeFieldFilter — never inline filter operators. (`model ListEventsParamsFilter { id?: Common.StringFieldFilter; customer_id?: Common.ULIDFieldFilter; time?: Common.DateTimeFieldFilter; }`)
**deepObject filter param with explode** — The filter query param uses @query(#{ style: "deepObject", explode: true }) so filter[field][op]=value query strings deserialize correctly server-side. (`@query(#{ style: "deepObject", explode: true }) filter?: ListEventsParamsFilter,`)
**202 ingest response with empty body** — IngestEventsResponse declares @statusCode _: 202 and no @body, reflecting async fire-and-forget processing. (`model IngestEventsResponse { @statusCode _: 202; }`)
**CloudEvents field constraints** — MeteringEvent follows the CloudEvents spec: id/source/type/subject have @minLength(1); specversion defaults to '1.0'; data is Record<unknown>; source has @format("uri-reference"). (`@minLength(1) @format("uri-reference") source: string;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `event.tsp` | MeteringEvent (CloudEvents schema), IngestedEvent (with ingested_at/stored_at/validation_errors), and IngestedEventValidationError (code+message+attributes). | New validation error codes added in the Go layer need corresponding documentation here; IngestedEvent.customer is a Shared.ResourceReference<Customers.Customer>. |
| `operations.tsp` | EventsOperations interface with cursor-paginated list (marked UnstableExtension) and three @sharedRoute POST ingest overloads, plus ListEventsParamsFilter and IngestEventsResponse models. | All three ingest overloads share the same @operationId; a fourth content-type overload must also use @sharedRoute and the same operationId. sort defaults to time desc; bare field defaults to desc. |
| `index.tsp` | Barrel import re-exporting event.tsp and operations.tsp. | Every new .tsp file added here must be imported in index.tsp. |

## Anti-Patterns

- Adding pagination to the ingest endpoint — ingest is fire-and-forget 202
- Using PagePaginationQuery for event listing — events use cursor (after/before) pagination
- Defining filter operators inline instead of using Common.*FieldFilter types
- Removing @sharedRoute from any ingest overload — collapses three content-type variants into one
- Adding response body fields to IngestEventsResponse — the ingest acknowledgment is intentionally empty

## Decisions

- **Three @sharedRoute ingest overloads instead of a single union body** — OpenAPI requires distinct requestBody entries per content-type for correct SDK generation; @sharedRoute expresses this without duplicating the operation ID.
- **Cursor pagination for events** — Event streams grow continuously and page-number pagination is unstable on append-only data; cursor-based (after/before) pagination gives stable traversal.

## Example: Add a new filter field to event listing

```
// In operations.tsp, add to ListEventsParamsFilter:
model ListEventsParamsFilter {
  // ... existing fields ...

  /** Filter events by validation error code. */
  validation_error_code?: Common.StringFieldFilterExact;
}
```

<!-- archie:ai-end -->
