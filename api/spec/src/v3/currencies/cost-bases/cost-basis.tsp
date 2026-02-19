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
   * The FIAT currency code for the cost basis. It should be a standard ISO 4217 code representing the currency in which the cost is calculated.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  fiat_code: Shared.CurrencyCode;

  /**
   * The cost rate for the currency. It is the amount of money that represents one unit of the currency.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  rate: Shared.Numeric;

  /**
   * An ISO-8601 timestamp representation of the date from which the cost basis is effective.
   * If not provided, it will be effective immediately and will set to time.Now() by the system.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  effective_from?: Shared.DateTime;

  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
