import "../shared/index.tsp";

namespace Meters;

/**
 * A meter is a configuration that defines how to match and aggregate events.
 */
@friendlyName("Meter")
@example(#{
  id: "01G65Z755AFWAKHE12NY0CQ9FH",
  key: "tokens_total",
  name: "Tokens Total",
  description: "AI Token Usage",
  aggregation: "sum",
  event_type: "prompt",
  value_property: "$.tokens",
  dimensions: #{ `model`: "$.model", type: "$.type" },
  created_at: Shared.DateTime.fromISO("2024-01-01T01:01:01.001Z"),
  updated_at: Shared.DateTime.fromISO("2024-01-01T01:01:01.001Z"),
})
model Meter {
  ...Shared.ResourceWithKey;

  /**
   * The aggregation type to use for the meter.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  aggregation: MeterAggregation;

  /**
   * The event type to include in the aggregation.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @minLength(1)
  @example("prompt")
  event_type: string;

  /**
   * The date since the meter should include events.
   * Useful to skip old events.
   * If not specified, all historical events are included.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  events_from?: Shared.DateTime;

  /**
   * JSONPath expression to extract the value from the ingested event's data property.
   *
   * The ingested value for sum, avg, min, and max aggregations is a number or a string that can be parsed to a number.
   *
   * For unique_count aggregation, the ingested value must be a string. For count aggregation the value_property is ignored.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @minLength(1)
  @example("$.tokens")
  value_property?: string;

  /**
   * Named JSONPath expressions to extract the group by values from the event data.
   *
   * Keys must be unique and consist only alphanumeric and underscore characters.
   *
   */
  // TODO: add key format enforcement
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @example(#{ type: "$.type" })
  dimensions?: Record<string>;
}

/**
 * The aggregation type to use for the meter.
 */
@friendlyName("MeterAggregation")
union MeterAggregation {
  sum: "sum",
  count: "count",
  unique_count: "unique_count",
  avg: "avg",
  min: "min",
  max: "max",
  latest: "latest",
}
