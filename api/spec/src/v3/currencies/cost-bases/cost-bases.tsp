import "../../shared/index.tsp";
import "./cost-basis.tsp";

namespace Currencies;

/**
 * Describes currency cost bases
 */
@friendlyName("BillingCostBases")
model CostBases is CostBasis[];
