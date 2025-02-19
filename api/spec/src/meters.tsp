import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

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
@extension("x-go-type", "models.Meter")
@extension(
  "x-go-type-import",
  {
    path: "github.com/openmeterio/openmeter/pkg/models",
  }
)
@friendlyName("Meter")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  slug: "tokens_total",
  description: "AI Token Usage",
  aggregation: MeterAggregation.SUM,
  windowSize: WindowSize.MINUTE,
  eventType: "prompt",
  valueProperty: "$.tokens",
  groupBy: #{ `model`: "$.model", type: "$.type" },
})
model Meter {
  /**
   * A unique identifier for the meter.
   */
  @visibility(Lifecycle.Read)
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  id: ULID;

  /**
   * A unique, human-readable identifier for the meter.
   * Must consist only alphanumeric and underscore characters.
   */
  @example("tokens_total")
  slug: Key;

  /**
   * A description of the meter.
   */
  @minLength(1)
  @example("AI Token Usage")
  description?: string;

  @example(MeterAggregation.SUM)
  aggregation: MeterAggregation;

  @example(WindowSize.MINUTE)
  windowSize: WindowSize;

  /**
   * The event type to aggregate.
   */
  @minLength(1)
  @example("prompt")
  eventType: string;

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
#suppress "@openmeter/api-spec/casing" "Use existing values"
@extension("x-go-type", "models.MeterAggregation")
@extension(
  "x-go-type-import",
  {
    path: "github.com/openmeterio/openmeter/pkg/models",
  }
)
@friendlyName("MeterAggregation")
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
#suppress "@openmeter/api-spec/casing" "Use existing values"
@extension("x-go-type", "models.WindowSize")
@extension(
  "x-go-type-import",
  {
    path: "github.com/openmeterio/openmeter/pkg/models",
  }
)
@friendlyName("WindowSize")
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
  from?: DateTime;
  to?: DateTime;
  windowSize?: WindowSize;
  data: MeterQueryRow[];
}

/**
 * A row in the result of a meter query.
 */
@extension("x-go-type", "models.MeterQueryRow")
@extension(
  "x-go-type-import",
  {
    path: "github.com/openmeterio/openmeter/pkg/models",
  }
)
@friendlyName("MeterQueryRow")
@example(#{
  value: 12,
  windowStart: DateTime.fromISO("2023-01-01T00:00:00Z"),
  windowEnd: DateTime.fromISO("2023-01-02T00:00:00Z"),
  subject: "customer-1",
  groupBy: #{ `model`: "gpt-4-turbo", type: "prompt" },
})
model MeterQueryRow {
  value: numeric;
  windowStart: DateTime;
  windowEnd: DateTime;
  subject?: string;
  groupBy?: Record<string>;
}
