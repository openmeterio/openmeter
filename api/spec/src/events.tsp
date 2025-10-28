import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/events")
@tag("Events")
@friendlyName("Events")
interface EventsEndpoints {
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
     * Client ID
     * Useful to track progress of a query.
     */
    @query
    @example("f74e58ed-94ce-4041-ae06-cf45420451a3")
    @minLength(1)
    @maxLength(36)
    clientId?: string,

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
     * The event ID.
     *
     * Accepts partial ID.
     */
    @query
    @example("my-event-id")
    id?: string,

    /**
     * The event subject.
     *
     * Accepts partial subject.
     */
    @query
    @example("customer-1")
    subject?: string,

    /**
     * The event customer ID.
     */
    @query(#{ explode: true })
    @example(#["01HZXABCD3M6E7K8J9PQRS1TUV", "01HZXABCDE3M6E7K8J9PQRSTUV"])
    customerId?: ULID[],

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
    @query
    @minValue(1)
    @maxValue(100)
    @example(100)
    limit?: integer = 100,
  ): IngestedEvent[] | CommonErrors;

  /**
   * Ingests an event or batch of events following the CloudEvents specification.
   */
  @post
  @operationId("ingestEvents")
  @summary("Ingest events")
  @sharedRoute
  ingestEvent(
    @header contentType: "application/cloudevents+json",
    @body body: Event,
  ): {
    @statusCode _: 204;
  } | CommonErrors;

  @post
  @operationId("ingestEvents")
  @summary("")
  @sharedRoute
  ingestEvents(
    @header contentType: "application/cloudevents-batch+json",
    @body body: Event[],
  ): {
    @statusCode _: 204;
  } | CommonErrors;

  @post
  @operationId("ingestEvents")
  @summary("")
  @sharedRoute
  ingestEventsJson(
    @header contentType: "application/json",
    @body body: IngestEventsBody,
  ): {
    @statusCode _: 204;
  } | CommonErrors;
}

@route("/api/v2/events")
@tag("Events")
@friendlyName("EventsV2")
interface EventsV2Endpoints {
  /**
   * List ingested events with advanced filtering and cursor pagination.
   */
  @list
  @operationId("listEventsV2")
  @summary("List ingested events")
  list(
    ...QueryCursorPagination,

    /**
     * Client ID
     * Useful to track progress of a query.
     */
    @query
    @example("f74e58ed-94ce-4041-ae06-cf45420451a3")
    @minLength(1)
    @maxLength(36)
    clientId?: string,

    /**
     * The filter for the events encoded as JSON string.
     */
    @example(#{
      id: #{ $eq: "my-event-id" },
      source: #{ $eq: "my-event-source" },
      subject: #{ $eq: "my-event-subject" },
      type: #{ $eq: "my-event-type" },
      time: #{
        $and: #[
          #{ $gte: DateTime.fromISO("2025-01-01T00:00:00Z") },
          #{ $lte: DateTime.fromISO("2025-01-02T00:00:00Z") }
        ],
      },
    })
    @query
    @encode("application/json")
    filter?: {
      id?: FilterString;
      source?: FilterString;
      subject?: FilterString;
      customerId?: FilterIDExact;
      type?: FilterString;
      time?: FilterTime;
      ingestedAt?: FilterTime;
    },
  ): CursorPaginatedResponse<IngestedEvent> | CommonErrors;
}

/**
 * The body of the events request.
 * Either a single event or a batch of events.
 */
@friendlyName("IngestEventsBody")
union IngestEventsBody {
  @friendlyName("Event")
  Event: Event,

  @friendlyName("Events")
  Events: Event[],
}

/**
 * CloudEvents Specification JSON Schema
 *
 * Optional properties are nullable according to the CloudEvents specification:
 * OPTIONAL not omitted attributes MAY be represented as a null JSON value.
 */
@extension("x-go-type", "event.Event")
@extension(
  "x-go-type-import",
  #{ path: "github.com/cloudevents/sdk-go/v2/event" }
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
   * Content type of the CloudEvents data value. Only the value "application/json" is allowed over HTTP.
   */
  @example("application/json")
  datacontenttype?: "application/json" | null;

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
  subject: string;

  /**
   * Timestamp of when the occurrence happened. Must adhere to RFC 3339.
   */
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  time?: DateTime | null;

  /**
   * The event payload.
   * Optional, if present it must be a JSON object.
   */
  @oneOf
  data?: Record<unknown> | null;
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
  /**
   * The original event ingested.
   */
  event: Event;

  /**
   * The customer ID if the event is associated with a customer.
   */
  customerId?: ULID;

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
