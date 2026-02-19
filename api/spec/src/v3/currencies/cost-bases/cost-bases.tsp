import "../../shared/index.tsp";
import "./cost-basis.tsp";

namespace Currencies;

/**
 * CostBases represents a list of cost basis for a currency. For custom currencies, there can be multiple cost bases with different effective_from dates. For standard currencies, there will be only one cost basis without an effective_from date.
 */
@friendlyName("BillingCostBases")
model CostBases {
  /**
   * List of cost basis for a currency. For custom currencies, there can be multiple cost bases with different effective_from dates. For standard currencies, there will be only one cost basis without an effective_from date.
   */
  data: CostBasis[];
}
