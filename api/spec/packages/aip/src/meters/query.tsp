import "@typespec/json-schema";
import "../shared/index.tsp";

using TypeSpec.JsonSchema;

namespace Meters;

/**
 * The granularity of the time grouping.
 * Time durations are specified in ISO 8601 format.
 */
@friendlyName("MeterQueryGranularity")
union MeterQueryGranularity {
  /**
   * 1 minute time buckets.
   */
  @summary("1 minute")
  Minute: "PT1M",

  /**
   * 1 hour time buckets.
   */
  @summary("1 hour")
  Hour: "PT1H",

  /**
   * 1 day time buckets.
   */
  @summary("1 day")
  Day: "P1D",

  /**
   * 1 month time buckets.
   */
  @summary("1 month")
  Month: "P1M",
}

/**
 * Filters to apply to a meter query.
 */
@friendlyName("MeterQueryFilters")
model MeterQueryFilters {
  /**
   * Filters to apply to the dimensions of the query.
   */
  @maxProperties(10)
  dimensions?: {
    /**
     * Filters to apply to the subject of the query.
     * Only equals ("eq", "in") comparisons are supported.
     */
    subject?: Shared.QueryFilterString;

    /**
     * Filters to apply to the customer ID of the query.
     * Only equals ("eq", "in") comparisons are supported.
     */
    customer_id?: Shared.QueryFilterString;

    /**
     * Additional dimensions to filter by.
     */
    ...Record<Shared.QueryFilterStringMapItem>;
  };
}

/**
 * A meter query request.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "keep the request flat"
@friendlyName("MeterQueryRequest")
@example(#{
  from: Shared.DateTime.fromISO("2023-01-01T00:00:00Z"),
  to: Shared.DateTime.fromISO("2023-01-02T00:00:00Z"),
  granularity: MeterQueryGranularity.Day,
  time_zone: "UTC",
})
model MeterQueryRequest {
  /**
   * The start of the period the usage is queried from.
   */
  from?: Shared.DateTime;

  /**
   * The end of the period the usage is queried to.
   */
  to?: Shared.DateTime;

  /**
   * The size of the time buckets to group the usage into.
   * If not specified, the usage is aggregated over the entire period.
   */
  granularity?: MeterQueryGranularity;

  /**
   * The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
   * The time zone is used to determine the start and end of the time buckets.
   * If not specified, the UTC timezone will be used.
   */
  time_zone?: string = "UTC";

  /**
   * The dimensions to group the results by.
   */
  @example(#["model", "type"])
  @maxItems(100)
  group_by_dimensions?: string[];

  /**
   * Filters to apply to the query.
   */
  filters?: MeterQueryFilters;
}

/**
 * Meter query result.
 */
@friendlyName("MeterQueryResult")
@example(#{
  from: Shared.DateTime.fromISO("2023-01-01T00:00:00Z"),
  to: Shared.DateTime.fromISO("2023-01-02T00:00:00Z"),
  data: #[
    #{
      value: "12.3456",
      from: Shared.DateTime.fromISO("2023-01-01T00:00:00Z"),
      to: Shared.DateTime.fromISO("2023-01-02T00:00:00Z"),
      dimensions: #{
        customer_id: "01G65Z755AFWAKHE12NY0CQ9FH",
        `model`: "gpt-4-turbo",
        type: "input",
      },
    }
  ],
})
model MeterQueryResult {
  /**
   * The start of the period the usage is queried from.
   */
  from?: Shared.DateTime;

  /**
   * The end of the period the usage is queried to.
   */
  to?: Shared.DateTime;

  /**
   * The usage data.
   * If no data is available, an empty array is returned.
   */
  data: MeterQueryRow[];
}

/**
 * A row in the result of a meter query.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "keep the row flat"
@friendlyName("MeterQueryRow")
@example(#{
  value: "12.3456",
  from: Shared.DateTime.fromISO("2023-01-01T00:00:00Z"),
  to: Shared.DateTime.fromISO("2023-01-02T00:00:00Z"),
  dimensions: #{
    customer_id: "01G65Z755AFWAKHE12NY0CQ9FH",
    `model`: "gpt-4-turbo",
    type: "input",
  },
})
model MeterQueryRow {
  /**
   * The aggregated value.
   */
  value: Shared.Numeric;

  /**
   * The start of the time bucket the value is aggregated over.
   */
  from: Shared.DateTime;

  /**
   * The end of the time bucket the value is aggregated over.
   */
  to: Shared.DateTime;

  /**
   * The dimensions the value is aggregated over.
   */
  dimensions: {
    /**
     * The subject the value is aggregated over.
     * If not specified, the value is aggregated over all subjects.
     */
    subject?: string;

    /**
     * The customer ID the value is aggregated over.
     */
    customer_id?: string;

    /**
     * Additional dimensions the value is aggregated over.
     */
    ...Record<string>;
  };
}
