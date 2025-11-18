import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace Events;

/**
 * CloudEvents event.
 */
@extension("x-go-type", "event.Event")
@extension(
  "x-go-type-import",
  #{ path: "github.com/cloudevents/sdk-go/v2/event" }
)
@friendlyName("UsageEvent")
@summary("Usage Event")
@example(#{
  id: "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
  source: "service-name",
  specversion: "1.0",
  type: "prompt",
  subject: "customer-id",
  time: Shared.DateTime.fromISO("2023-01-01T01:01:01.001Z"),
  data: #{
    prompt: "Hello, world!",
    tokens: 100,
    `model`: "gpt-4o",
    type: "input",
  },
})
model UsageEvent {
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
  @example(Shared.DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  time?: Shared.DateTime | null;

  /**
   * The event payload.
   * Optional, if present it must be a flat JSON object with string, integer, or float values.
   */
  data?: Record<string | integer | float | null> | null;
}
