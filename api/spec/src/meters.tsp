import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./types.tsp";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/meters")
@tag("Meters")
interface Meters {
  /**
   * List meters.
   */
  @get
  @operationId("listMeters")
  @summary("List meters")
  list(): Meter[] | CommonErrors;

  /**
   * Get a meter by ID or slug.
   */
  @get
  @operationId("getMeter")
  @summary("Get meter")
  get(
    @path meterIdOrSlug: MeterIdentifier,
  ): Meter | NotFoundError | CommonErrors;

  /**
   * Create a meter.
   */
  @post
  @operationId("createMeter")
  @summary("Create meter")
  create(
    @body
    meter: MeterCreate,
  ): {
    @statusCode _: 201;
    @body body: OpenMeter.Meter;
  } | OpenMeter.CommonErrors;

  /**
   * Delete a meter.
   */
  @delete
  @operationId("deleteMeter")
  @summary("Delete meter")
  delete(
    @path meterIdOrSlug: OpenMeter.MeterIdentifier,
  ): void | OpenMeter.CommonErrors;

  /**
   * Query meter for usage.
   */
  @get
  @route("/{meterIdOrSlug}/query")
  @operationId("queryMeter")
  @summary("Query meter")
  @sharedRoute
  queryJson(@path meterIdOrSlug: MeterIdentifier, ...MeterQuery): {
    @header contentType: "application/json";
    @body _: MeterQueryResult;
  } | NotFoundError | CommonErrors;

  /**
   * Query meter for usage.
   */
  @get
  @route("/{meterIdOrSlug}/query")
  @operationId("queryMeter")
  @summary("Query meter")
  @sharedRoute
  queryCsv(@path meterIdOrSlug: MeterIdentifier, ...MeterQuery): {
    @header contentType: "text/csv";

    @body
    @example("""
      window_start,window_end,subject,model,type,value
      2023-01-01T00:00:00Z,2023-01-01T00:01:00Z,customer_1,gpt-4-turbo,input,12
      2023-01-01T00:01:00Z,2023-01-02T00:02:00Z,customer_1,gpt-4-turbo,input,20
      2023-01-01T00:02:00Z,2023-01-02T00:03:00Z,customer_2,gpt-4-turbo,output,4
      """)
    _: string;
  } | NotFoundError | CommonErrors;

  /**
   * List subjects for a meter.
   */
  @get
  @route("/{meterIdOrSlug}/subjects")
  @operationId("listMeterSubjects")
  @summary("List meter subjects")
  listSubjects(@path meterIdOrSlug: MeterIdentifier): {
    @body
    @example(#["customer_1", "customer_2"])
    _: string[];
  } | CommonErrors;
}

/**
 * A meter is a configuration that defines how to match and aggregate events.
 */
@friendlyName("MeterCreate")
@example(#{
  slug: "tokens_total",
  name: "Tokens Total",
  description: "AI Token Usage",
  aggregation: OpenMeter.MeterAggregation.SUM,
  eventType: "prompt",
  valueProperty: "$.tokens",
  groupBy: #{ `model`: "$.model", type: "$.type" },
})
model MeterCreate is Rest.Resource.ResourceCreateModel<OpenMeter.Meter>;

/**
 * A meter is a configuration that defines how to match and aggregate events.
 */
@friendlyName("Meter")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  slug: "tokens_total",
  name: "Tokens Total",
  description: "AI Token Usage",
  aggregation: MeterAggregation.SUM,
  eventType: "prompt",
  valueProperty: "$.tokens",
  groupBy: #{ `model`: "$.model", type: "$.type" },
  createdAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updatedAt: DateTime.fromISO("2024-01-01T01:01:01.001Z"),
})
model Meter {
  // We omit name from the global.Resource as name is optional in the Meter model to avoiud breaking changes for now.
  ...OmitProperties<global.Resource, "name">;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   * Defaults to the slug if not specified.
   */
  @summary("Display name")
  @minLength(1)
  @maxLength(256)
  name?: string;

  /**
   * A unique, human-readable identifier for the meter.
   * Must consist only alphanumeric and underscore characters.
   */
  @example("tokens_total")
  slug: Key;

  /**
   * The aggregation type to use for the meter.
   */
  @example(MeterAggregation.SUM)
  aggregation: MeterAggregation;

  /**
   * The event type to aggregate.
   */
  @minLength(1)
  @example("prompt")
  eventType: string;

  /**
   * The date since the meter should include events.
   * Useful to skip old events.
   * If not specified, all historical events are included.
   */
  eventFrom?: DateTime;

  /**
   * JSONPath expression to extract the value from the ingested event's data property.
   *
   * The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be parsed to a number.
   *
   * For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the valueProperty is ignored.
   */
  @minLength(1)
  @example("$.tokens")
  valueProperty?: string;

  /**
   * Named JSONPath expressions to extract the group by values from the event data.
   *
   * Keys must be unique and consist only alphanumeric and underscore characters.
   *
   * TODO: add key format enforcement
   */
  @example(#{ type: "$.type" })
  groupBy?: Record<string>;
}

/**
 * The aggregation type to use for the meter.
 */
@friendlyName("MeterAggregation")
@extension(
  "x-enum-varnames",
  ["Sum", "Count", "UniqueCount", "Avg", "Min", "Max"]
)
enum MeterAggregation {
  SUM,
  COUNT,
  UNIQUE_COUNT,
  AVG,
  MIN,
  MAX,
}

/**
 * Aggregation window size.
 */
@friendlyName("WindowSize")
@extension("x-enum-varnames", ["Minute", "Hour", "Day"])
enum WindowSize {
  MINUTE,
  HOUR,
  DAY,
}

/**
 * A unique meter identifier.
 */
alias MeterIdentifier = ULIDOrKey;

/**
 * Meter query parameters.
 */
@friendlyName("MeterQuery")
model MeterQuery {
  /**
   * Start date-time in RFC 3339 format.
   *
   * Inclusive.
   */
  @query(#{ explode: true })
  from?: DateTime;

  /**
   * End date-time in RFC 3339 format.
   *
   * Inclusive.
   */
  @query(#{ explode: true })
  to?: DateTime;

  /**
   * If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
   */
  @query
  windowSize?: WindowSize;

  /**
   * The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
   * If not specified, the UTC timezone will be used.
   */
  @query
  @example("America/New_York")
  windowTimeZone?: string = "UTC";

  /**
   * Filtering by multiple subjects.
   */
  @query(#{ explode: true })
  @example(#["customer-1", "customer-2"])
  subject?: string[];

  /**
   * Simple filter for group bys with exact match.
   */
  @query(#{ explode: true, style: "deepObject" })
  @example(#{ `model`: "gpt-4-turbo", type: "prompt" })
  filterGroupBy?: Record<string>;

  /**
   * If not specified a single aggregate will be returned for each subject and time window.
   * `subject` is a reserved group by value.
   */
  @query(#{ explode: true })
  @example(#["model", "type"])
  groupBy?: string[];
}

/**
 * The result of a meter query.
 */
@friendlyName("MeterQueryResult")
@example(#{
  from: DateTime.fromISO("2023-01-01T00:00:00Z"),
  to: DateTime.fromISO("2023-01-02T00:00:00Z"),
  windowSize: WindowSize.DAY,
  data: #[
    #{
      value: 12,
      windowStart: DateTime.fromISO("2023-01-01T00:00:00Z"),
      windowEnd: DateTime.fromISO("2023-01-02T00:00:00Z"),
      subject: "customer-1",
      groupBy: #{ `model`: "gpt-4-turbo", type: "prompt" },
    }
  ],
})
model MeterQueryResult {
  /**
   * The start of the period the usage is queried from.
   * If not specified, the usage is queried from the beginning of time.
   */
  from?: DateTime;

  /**
   * The end of the period the usage is queried to.
   * If not specified, the usage is queried up to the current time.
   */
  to?: DateTime;

  /**
   * The window size that the usage is aggregated.
   * If not specified, the usage is aggregated over the entire period.
   */
  windowSize?: WindowSize;

  /**
   * The usage data.
   * If no data is available, an empty array is returned.
   */
  data: MeterQueryRow[];
}

/**
 * A row in the result of a meter query.
 */
@friendlyName("MeterQueryRow")
@example(#{
  value: 12,
  windowStart: DateTime.fromISO("2023-01-01T00:00:00Z"),
  windowEnd: DateTime.fromISO("2023-01-02T00:00:00Z"),
  subject: "customer-1",
  groupBy: #{ `model`: "gpt-4-turbo", type: "prompt" },
})
model MeterQueryRow {
  /**
   * The aggregated value.
   */
  value: float64;

  /**
   * The start of the window the value is aggregated over.
   */
  windowStart: DateTime;

  /**
   * The end of the window the value is aggregated over.
   */
  windowEnd: DateTime;

  /**
   * The subject the value is aggregated over.
   * If not specified, the value is aggregated over all subjects.
   */
  subject: string | null;

  /**
   * The group by values the value is aggregated over.
   */
  groupBy: Record<string | null>;
}
