import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "./types.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Query params for listing meters.
 */
@friendlyName("queryMeterList")
model ListMetersParams {
  ...QueryPagination;

  /**
   * Order meters by.
   * @TODO: set default value to MeterOrderBy.key
   */
  ...QueryOrdering<MeterOrderBy>;

  /**
   * Include deleted meters.
   */
  @query
  includeDeleted?: boolean = false;
}

/**
 * Order by options for meters.
 */
@friendlyName("MeterOrderBy")
enum MeterOrderBy {
  Key: "key",
  Name: "name",
  Aggregation: "aggregation",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  CreatedAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  UpdatedAt: "updatedAt",
}

@route("/api/v1/meters")
@tag("Meters")
@friendlyName("Meters")
interface MetersEndpoints {
  /**
   * List meters.
   */
  @get
  @operationId("listMeters")
  @summary("List meters")
  list(...ListMetersParams): Meter[] | CommonErrors;

  /**
   * Get a meter by ID or slug.
   */
  @get
  @route("/{meterIdOrSlug}")
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
    @body body: Meter;
  } | CommonErrors;

  /**
   * Update a meter.
   */
  @put
  @route("/{meterIdOrSlug}")
  @operationId("updateMeter")
  @summary("Update meter")
  update(
    @path meterIdOrSlug: MeterIdentifier,

    @body
    meter: MeterUpdate,
  ): {
    @body body: Meter;
  } | CommonErrors;

  /**
   * Delete a meter.
   */
  @delete
  @route("/{meterIdOrSlug}")
  @operationId("deleteMeter")
  @summary("Delete meter")
  delete(@path meterIdOrSlug: MeterIdentifier): void | CommonErrors;

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

  #suppress "@openmeter/api-spec/operationSummary" "Avoid duplicating the summary in OpenAPI yaml"
  @get
  @route("/{meterIdOrSlug}/query")
  @operationId("queryMeter")
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

  @post
  @route("/{meterIdOrSlug}/query")
  @operationId("queryMeterPost")
  @summary("Query meter")
  @sharedRoute
  query(
    @path meterIdOrSlug: MeterIdentifier,
    @body request: MeterQueryRequest,
  ): {
    @header contentType: "application/json";
    @body _: MeterQueryResult;
  } | NotFoundError | CommonErrors;

  /**
   * List subjects for a meter.
   */
  @get
  @route("/{meterIdOrSlug}/subjects")
  @operationId("listMeterSubjects")
  @summary("List meter subjects")
  listSubjects(
    @path meterIdOrSlug: MeterIdentifier,

    /**
     * Start date-time in RFC 3339 format.
     *
     * Inclusive. Defaults to the beginning of time.
     *
     * For example: ?from=2025-01-01T00%3A00%3A00.000Z
     */
    @query(#{ explode: true })
    from?: DateTime,

    /**
     * End date-time in RFC 3339 format.
     *
     * Inclusive.
     *
     * For example: ?to=2025-02-01T00%3A00%3A00.000Z
     */
    @query(#{ explode: true })
    to?: DateTime,
  ): {
    @body
    @example(#["customer_1", "customer_2"])
    _: string[];
  } | CommonErrors;

  /**
   * List meter group by values.
   */
  @get
  @route("/{meterIdOrSlug}/group-by/{groupByKey}/values")
  @operationId("listMeterGroupByValues")
  @summary("List meter group by values")
  listGroupByValues(
    @path meterIdOrSlug: MeterIdentifier,
    @path groupByKey: string,

    /**
     * Start date-time in RFC 3339 format.
     *
     * Inclusive. Defaults to 24 hours ago.
     *
     * For example: ?from=2025-01-01T00%3A00%3A00.000Z
     */
    @query(#{ explode: true })
    from?: DateTime,

    /**
     * End date-time in RFC 3339 format.
     *
     * Inclusive.
     *
     * For example: ?to=2025-02-01T00%3A00%3A00.000Z
     */
    @query(#{ explode: true })
    to?: DateTime,
  ): {
    @body
    @example(#["gpt-4-turbo", "gpt-4o"])
    _: string[];
  } | CommonErrors;
}

/**
 * A meter create model.
 */
@friendlyName("MeterCreate")
@example(#{
  slug: "tokens_total",
  name: "Tokens Total",
  description: "AI Token Usage",
  aggregation: MeterAggregation.SUM,
  eventType: "prompt",
  valueProperty: "$.tokens",
  groupBy: #{ `model`: "$.model", type: "$.type" },
})
model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;

/**
 * A meter update model.
 *
 * Only the properties that can be updated are included.
 * For example, the slug and aggregation cannot be updated.
 */
@friendlyName("MeterUpdate")
@example(#{
  name: "Tokens Total",
  description: "AI Token Usage",
  groupBy: #{ `model`: "$.model", type: "$.type" },
})
model MeterUpdate is TypeSpec.Rest.Resource.ResourceReplaceModel<Meter>;

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
  // We omit name from the global.Resource as name is optional in the Meter model to avoid breaking changes for now.
  ...OmitProperties<global.Resource, "name">;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   * Defaults to the slug if not specified.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Display name")
  @minLength(1)
  @maxLength(256)
  name?: string;

  /**
   * A unique, human-readable identifier for the meter.
   * Must consist only alphanumeric and underscore characters.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @example("tokens_total")
  slug: Key;

  /**
   * The aggregation type to use for the meter.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @example(MeterAggregation.SUM)
  aggregation: MeterAggregation;

  /**
   * The event type to aggregate.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @minLength(1)
  @example("prompt")
  eventType: string;

  /**
   * The date since the meter should include events.
   * Useful to skip old events.
   * If not specified, all historical events are included.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  eventFrom?: DateTime;

  /**
   * JSONPath expression to extract the value from the ingested event's data property.
   *
   * The ingested value for SUM, AVG, MIN, and MAX aggregations is a number or a string that can be parsed to a number.
   *
   * For UNIQUE_COUNT aggregation, the ingested value must be a string. For COUNT aggregation the valueProperty is ignored.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @minLength(1)
  @example("$.tokens")
  valueProperty?: string;

  /**
   * Named JSONPath expressions to extract the group by values from the event data.
   *
   * Keys must be unique and consist only alphanumeric and underscore characters.
   *
   */
  // TODO: add key format enforcement
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @example(#{ type: "$.type" })
  groupBy?: Record<string>;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: Annotations | null;
}

/**
 * The aggregation type to use for the meter.
 */
@friendlyName("MeterAggregation")
@extension(
  "x-enum-varnames",
  #["Sum", "Count", "UniqueCount", "Avg", "Min", "Max", "Latest"]
)
enum MeterAggregation {
  SUM,
  COUNT,
  UNIQUE_COUNT,
  AVG,
  MIN,
  MAX,
  LATEST,
}

/**
 * Aggregation window size.
 */
@friendlyName("WindowSize")
@extension("x-enum-varnames", #["Minute", "Hour", "Day", "Month"])
enum WindowSize {
  MINUTE,
  HOUR,
  DAY,
  MONTH,
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
   * Client ID
   * Useful to track progress of a query.
   */
  @query
  @example("f74e58ed-94ce-4041-ae06-cf45420451a3")
  @minLength(1)
  @maxLength(36)
  clientId?: string;

  /**
   * Start date-time in RFC 3339 format.
   *
   * Inclusive.
   *
   * For example: ?from=2025-01-01T00%3A00%3A00.000Z
   */
  @query(#{ explode: true })
  from?: DateTime;

  /**
   * End date-time in RFC 3339 format.
   *
   * Inclusive.
   *
   * For example: ?to=2025-02-01T00%3A00%3A00.000Z
   */
  @query(#{ explode: true })
  to?: DateTime;

  /**
   * If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
   *
   * For example: ?windowSize=DAY
   */
  @query
  windowSize?: WindowSize;

  /**
   * The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
   * If not specified, the UTC timezone will be used.
   *
   * For example: ?windowTimeZone=UTC
   */
  @query
  @example("America/New_York")
  windowTimeZone?: string = "UTC";

  /**
   * Filtering by multiple subjects.
   *
   * For example: ?subject=subject-1&subject=subject-2
   */
  @query(#{ explode: true })
  @example(#["subject-1", "subject-2"])
  subject?: string[];

  /**
   * Filtering by multiple customers.
   *
   * For example: ?filterCustomerId=customer-1&filterCustomerId=customer-2
   */
  @query(#{ explode: true })
  @maxItems(100)
  @example(#["customer-1", "customer-2"])
  filterCustomerId?: string[];

  /**
   * Simple filter for group bys with exact match.
   *
   * For example: ?filterGroupBy[vendor]=openai&filterGroupBy[model]=gpt-4-turbo
   *
   * ⚠️ __Deprecated__: Use `advancedMeterGroupByFilters` instead
   */
  #deprecated "Use advancedMeterGroupByFilters instead"
  @query(#{ explode: true, style: "deepObject" })
  filterGroupBy?: Record<string>;

  /**
   * Optional advanced meter group by filters.
   * You can use this to filter for values of the meter groupBy fields.
   */
  @query
  @summary("Advanced meter group by filters")
  @example(#{
    `model`: #{ $in: #["gpt-4", "gpt-4o"] },
    type: #{ $eq: "input" },
  })
  @encode("application/json")
  advancedMeterGroupByFilters?: Record<FilterString>;

  /**
   * If not specified a single aggregate will be returned for each subject and time window.
   * `subject` is a reserved group by value.
   *
   * For example: ?groupBy=subject&groupBy=model
   */
  @query(#{ explode: true })
  @example(#["model", "type"])
  groupBy?: string[];
}

/**
 * A meter query request.
 */
@friendlyName("MeterQueryRequest")
model MeterQueryRequest {
  /**
   * Client ID
   * Useful to track progress of a query.
   */
  @example("f74e58ed-94ce-4041-ae06-cf45420451a3")
  @minLength(1)
  @maxLength(36)
  clientId?: string;

  /**
   * Start date-time in RFC 3339 format.
   *
   * Inclusive.
   */
  from?: DateTime;

  /**
   * End date-time in RFC 3339 format.
   *
   * Inclusive.
   */
  to?: DateTime;

  /**
   * If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
   */
  @example(WindowSize.DAY)
  windowSize?: WindowSize;

  /**
   * The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
   * If not specified, the UTC timezone will be used.
   */
  @example("UTC")
  windowTimeZone?: string = "UTC";

  /**
   * Filtering by multiple subjects.
   */
  @example(#["subject-1", "subject-2"])
  @maxItems(100)
  subject?: string[];

  /**
   * Filtering by multiple customers.
   */
  @example(#["id-1", "id-2"])
  @maxItems(100)
  filterCustomerId?: string[];

  /**
   * Simple filter for group bys with exact match.
   */
  @example(#{ `model`: #["gpt-4-turbo", "gpt-4o"], type: #["prompt"] })
  filterGroupBy?: Record<string[]>;

  /**
   * Optional advanced meter group by filters.
   * You can use this to filter for values of the meter groupBy fields.
   */
  @example(#{
    `model`: #{ $in: #["gpt-4", "gpt-4o"] },
    type: #{ $eq: "input" },
  })
  advancedMeterGroupByFilters?: Record<FilterString>;

  /**
   * If not specified a single aggregate will be returned for each subject and time window.
   * `subject` is a reserved group by value.
   */
  @example(#["model", "type"])
  @maxItems(100)
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
   * The customer ID the value is aggregated over.
   */
  customerId?: string;

  /**
   * The group by values the value is aggregated over.
   */
  groupBy: Record<string | null>;
}
