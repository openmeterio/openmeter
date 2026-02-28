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

  /**
   * The ID of the custom currency this cost basis belongs to.
   */
  @visibility(Lifecycle.Read)
  currency_id: Shared.ULID;

  /**
   * The fiat currency code for the cost basis.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  fiat_code: Shared.CurrencyCode;

  /**
   * The cost rate for the currency.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  rate: Shared.Numeric;

  /**
   * An ISO-8601 timestamp representation of the date from which the cost basis is effective.
   * If not provided, it will be effective immediately and will set to `now` by the system.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  effective_from?: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
