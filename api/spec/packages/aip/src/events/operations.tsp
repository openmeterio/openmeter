import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./event.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Events;

/**
 * The events have been ingested and are being processed asynchronously.
 */
@friendlyName("IngestEventsResponse")
model IngestEventsResponse {
  @statusCode _: 202;
}

/**
 * Filter options for listing ingested events.
 */
@friendlyName("ListEventsParamsFilter")
model ListEventsParamsFilter {
  /**
   * Filter events by ID.
   */
  id?: Common.StringFieldFilter;

  /**
   * Filter events by source.
   */
  source?: Common.StringFieldFilter;

  /**
   * Filter events by subject.
   */
  subject?: Common.StringFieldFilter;

  /**
   * Filter events by type.
   */
  type?: Common.StringFieldFilter;

  /**
   * Filter events by the associated customer ID.
   */
  customer_id?: Common.ULIDFieldFilter;

  /**
   * Filter events by event time.
   */
  time?: Common.DateTimeFieldFilter;

  /**
   * Filter events by the time the event was ingested.
   */
  ingested_at?: Common.DateTimeFieldFilter;

  /**
   * Filter events by the time the event was stored.
   */
  stored_at?: Common.DateTimeFieldFilter;
}

interface EventsOperations {
  /**
   * List ingested events.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-metering-events")
  @summary("List metering events")
  list(
    ...Common.CursorPaginationQuery,

    /**
     * Filter events returned in the response.
     *
     * To filter events by subject add the following query param: filter[subject][eq]=customer-1
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListEventsParamsFilter,

    /**
     * Sort events returned in the response.
     * Supported sort attributes are:
     * - `time` (default)
     * - `ingested_at`
     * - `stored_at`
     *
     * When omitted, events are sorted by `time desc` (most recent first).
     * When a sort field is provided without a suffix, it sorts descending.
     * Append the `asc` suffix to sort ascending, or the `desc` suffix to sort descending.
     */
    @query
    sort?: Common.SortQuery,
  ): Shared.CursorPaginatedResponse<IngestedEvent> | Common.ErrorResponses;

  /**
   * Ingests an event or batch of events following the CloudEvents specification.
   */
  @post
  @operationId("ingest-metering-events")
  @summary("Ingest metering events")
  @sharedRoute
  ingestEvent(
    @header contentType: "application/cloudevents+json",
    @body body: MeteringEvent,
  ): IngestEventsResponse | Common.ErrorResponses;

  @post
  @operationId("ingest-metering-events")
  @summary("")
  @sharedRoute
  ingestEvents(
    @header contentType: "application/cloudevents-batch+json",
    @body body: MeteringEvent[],
  ): IngestEventsResponse | Common.ErrorResponses;

  @post
  @operationId("ingest-metering-events")
  @summary("")
  @sharedRoute
  ingestEventsJson(
    @header contentType: "application/json",
    @body body: MeteringEvent | MeteringEvent[],
  ): IngestEventsResponse | Common.ErrorResponses;
}
