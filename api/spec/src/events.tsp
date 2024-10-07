import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/events")
@tag("Events")
interface Events {
  /**
   * List ingested events within a time range.
   *
   * If the from query param is not provided it defaults to last 72 hours.
   */
  @get
  @operationId("listEvents")
  @summary("List ingested events")
  list(
    /**
     * Start date-time in RFC 3339 format.
     *
     * Inclusive.
     */
    @query(#{ explode: true })
    @example(utcDateTime.fromISO("2023-01-01T00:00:00Z"))
    ingestedAtFrom?: utcDateTime,

    /**
     * End date-time in RFC 3339 format.
     *
     * Inclusive.
     */
    @query(#{ explode: true })
    @example(utcDateTime.fromISO("2023-01-01T00:00:00Z"))
    ingestedAtTo?: utcDateTime,

    /**
     * If not provided lists all events.
     *
     * If provided with true, only list events with processing error.
     *
     * If provided with false, only list events without processing error.
     */
    @query(#{ explode: true })
    @example(false)
    hasError?: boolean,

    /**
     * The event ID.
     *
     * Accepts partial ID.
     */
    @query(#{ explode: true })
    @example("my-event-id")
    id?: string,

    /**
     * The event subject.
     *
     * Accepts partial subject.
     */
    @query(#{ explode: true })
    @example("customer-1")
    subject?: string,

    /**
     * Start date-time in RFC 3339 format.
     *
     * Inclusive.
     */
    @query(#{ explode: true })
    @example(utcDateTime.fromISO("2023-01-01T00:00:00Z"))
    from?: utcDateTime,

    /**
     * End date-time in RFC 3339 format.
     *
     * Inclusive.
     */
    @query(#{ explode: true })
    @example(utcDateTime.fromISO("2023-01-01T00:00:00Z"))
    to?: utcDateTime,

    /**
     * Number of events to return.
     */
    @query(#{ explode: true })
    @minValue(1)
    @maxValue(100)
    @example(100)
    limit?: integer = 100,
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
@friendlyName("Event")
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
@friendlyName("IngestedEvent")
@example(#{
  event: #{
    id: "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
    source: "service-name",
    specversion: "1.0",
    type: "prompt",
    subject: "customer-id",
    time: DateTime.fromISO("2023-01-01T01:01:01.001Z"),
  },
  ingestedAt: DateTime.fromISO("2023-01-01T01:01:01.001Z"),
  storedAt: DateTime.fromISO("2023-01-01T01:01:02.001Z"),
})
model IngestedEvent {
  /*
   * The original event ingested.
   */
  event: Event;

  /**
   * The validation error if the event failed validation.
   */
  validationError?: string;

  /**
   * The date and time the event was ingested.
   */
  ingestedAt: DateTime;

  /**
   * The date and time the event was stored.
   */
  storedAt: DateTime;
}
