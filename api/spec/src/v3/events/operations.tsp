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

interface EventsOperations {
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
