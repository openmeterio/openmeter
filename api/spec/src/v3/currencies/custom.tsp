import "../shared/index.tsp";

namespace Currencies;

/**
 * CustomCurrency describes a custom currency by the billing system
 */
@friendlyName("BillingCustomCurrency")
model CustomCurrency {
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * Custom currency code based on Standard ISO 4217 code.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  code: Shared.CurrencyCode;

  /**
   * Custom Currency name
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Custom Currency symbol
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  symbol: string;

  /**
   * Smallest denomination of the custom currency
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  smallest_denomination: uint8;

  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
