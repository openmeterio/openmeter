import "@typespec/json-schema";
import "../shared/index.tsp";
import "../meters/query.tsp";

namespace Features;

/**
 * A row in the result of a feature cost query.
 */
@friendlyName("FeatureCostQueryRow")
model FeatureCostQueryRow {
  /**
   * The metered usage value for the period.
   */
  usage: Shared.Numeric;

  /**
   * The computed cost amount (usage × unit cost).
   * Null when pricing is not available for the given combination of dimensions.
   */
  #suppress "@openmeter/api-spec-aip/no-nullable" "cost is nullable"
  cost: Shared.Numeric | null;

  /**
   * The currency code of the cost amount.
   */
  currency: Shared.CurrencyCode;

  /**
   * Detail message when cost amount is null, explaining why the cost could not be resolved.
   */
  detail?: string;

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

/**
 * Result of a feature cost query.
 */
@friendlyName("FeatureCostQueryResult")
model FeatureCostQueryResult {
  /**
   * Start of the queried period.
   */
  from?: Shared.DateTime;

  /**
   * End of the queried period.
   */
  to?: Shared.DateTime;

  /**
   * The cost data rows.
   */
  data: FeatureCostQueryRow[];
}
