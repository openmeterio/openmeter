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

interface UsageEventsOperations {
  /**
   * Ingests an event or batch of events following the CloudEvents specification.
   */
  @post
  @operationId("ingestUsageEvents")
  @summary("Ingest usage events")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @sharedRoute
  ingestEvent(
    @header contentType: "application/cloudevents+json",
    @body body: UsageEvent,
  ): {
    @statusCode _: 204;
  } | Common.ErrorResponses;

  @post
  @operationId("ingestUsageEvents")
  @summary("")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @sharedRoute
  ingestEvents(
    @header contentType: "application/cloudevents-batch+json",
    @body body: UsageEvent[],
  ): {
    @statusCode _: 204;
  } | Common.ErrorResponses;

  @post
  @operationId("ingestUsageEvents")
  @summary("")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @sharedRoute
  ingestEventsJson(
    @header contentType: "application/json",
    @body body: UsageEvent | UsageEvent[],
  ): {
    @statusCode _: 204;
  } | Common.ErrorResponses;
}
