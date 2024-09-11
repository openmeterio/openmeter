import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";
import "./date.tsp";
import "./errors.tsp";

using Http;
using OpenAPI;

namespace OpenMeter;

/**
 * CloudEvents Specification JSON Schema
 */
@extension("x-go-type", "event.Event")
@extension(
  "x-go-type-import",
  {
    path: "github.com/cloudevents/sdk-go/v2/event",
  }
)
@example(#{
  id: "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
  source: "service-name",
  specversion: "1.0",
  type: "prompt",
  subject: "customer-id",
  time: DateTime.fromISO("2023-01-01T01:01:01.001Z"),
})
model Event {
  /**
   * Identifies the event.
   */
  @minLength(1)
  @example("5c10fade-1c9e-4d6c-8275-c52c36731d3c")
  id: string;

  /**
   * Identifies the context in which an event happened.
   */
  @minLength(1)
  @format("uri-reference")
  @example("service-name")
  source: string;

  /**
   * The version of the CloudEvents specification which the event uses.
   */
  @minLength(1)
  @example("1.0")
  specversion: string = "1.0";

  /**
   * Contains a value describing the type of event related to the originating occurrence.
   */
  @minLength(1)
  @example("com.example.someevent")
  type: string;

  /**
   * Content type of the data value. Must adhere to RFC 2046 format.
   */
  @minLength(1)
  @example("application/json")
  datacontenttype?: string | null;

  /**
   * Identifies the schema that data adheres to.
   */
  @minLength(1)
  dataschema?: url | null;

  /**
   * Describes the subject of the event in the context of the event producer (identified by source).
   */
  @minLength(1)
  @example("customer-id")
  subject?: string;

  /**
   * Timestamp of when the occurrence happened. Must adhere to RFC 3339.
   */
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  time?: DateTime | null;

  /**
   * The event payload.
   */
  @oneOf
  data?: {} | string | null;
}

/**
 * An ingested event with optional validation error.
 */
model IngestedEvent {
  event: Event;
  validationError?: string;

  @format("date-time")
  @doc("Start date-time in RFC 3339 format. Inclusive.")
  @example("2023-01-01T00:00:00Z")
  ingestedAt: string;

  @format("date-time")
  @doc("Start date-time in RFC 3339 format. Inclusive.")
  @example("2023-01-01T00:00:00Z")
  storedAt: string;
}

/**
 * Query parameters for listing events.
 */
model EventQuery {
  @query
  @format("date-time")
  @doc("Start date-time in RFC 3339 format. Inclusive.")
  @example("2023-01-01T00:00:00Z")
  from: string;

  @query
  @format("date-time")
  @doc("End date-time in RFC 3339 format. Inclusive.")
  @example("2023-01-01T00:00:00Z")
  to: string;

  @query
  @minValue(1)
  @maxValue(100)
  @example(100)
  @doc("Number of events to return.")
  limit: integer = 100;
}

@route("/api/v1/events")
@tag("Events")
interface Events {
  @get
  @operationId("listEvents")
  @doc("List ingested events")
  list(
    @query
    @format("date-time")
    @doc("Start date-time in RFC 3339 format. Inclusive.")
    @example("2023-01-01T00:00:00Z")
    ingestedAtFrom: string,

    @query
    @format("date-time")
    @doc("End date-time in RFC 3339 format. Inclusive.")
    @example("2023-01-01T00:00:00Z")
    ingestedAtTo: string,

    @query
    @doc("If not provided lists all events. If provided with true, only list events with processing error. If provided with false, only list events without processing error.")
    @example(false)
    hasError: boolean,

    @query
    @doc("The event ID. Accepts partial ID.")
    @example("my-event-id")
    id: string,

    @query
    @doc("The event subject. Accepts partial subject.")
    @example("customer-1")
    subject: string,

    ...EventQuery,
  ): IngestedEvent[] | CommonErrors;

  @post
  @operationId("ingestEvents")
  @sharedRoute
  ingestEvent(@header contentType: "application/cloudevents+json", @body body: Event): {
    @header contentType: "application/cloudevents+json";
    @body _: Event;
  } | CommonErrors;

  @post
  @operationId("ingestEvents")
  @sharedRoute
  ingestEvents(@header contentType: "application/cloudevents-batch+json", @body body: Event[]): {
    @header contentType: "application/cloudevents-batch+json";
    @body _: Event[];
  } | CommonErrors;
}
