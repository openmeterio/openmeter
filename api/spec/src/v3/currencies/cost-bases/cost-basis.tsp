import "../../shared/index.tsp";

namespace Currencies;

/**
 * Describes currency basis supported by billing system.
 */
@friendlyName("BillingCostBasis")
model CostBasis {
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  @visibility(Lifecycle.Create, Lifecycle.Read)
  fiat_code: string;

  @visibility(Lifecycle.Create, Lifecycle.Read)
  rate: Shared.Numeric;

  /**
   * An ISO-8601 timestamp representation of the date from which the cost basis is effective.
   * If not provided, it will be effective immediately and will set to time.Now() by the system.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  effective_from?: Shared.DateTime;

  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
