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

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Create, Lifecycle.Read)
  currency_id: Shared.ULID;

  @visibility(Lifecycle.Create, Lifecycle.Read)
  fiat_code: string;

  @visibility(Lifecycle.Create, Lifecycle.Read)
  rate: numeric;

  @visibility(Lifecycle.Create, Lifecycle.Read)
  effective_from?: Shared.DateTime;

  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
