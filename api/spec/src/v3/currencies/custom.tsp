import "../shared/index.tsp";

namespace Currencies;

/**
 * CustomCurrency describes a custom currency by the billing system
 */
@friendlyName("BillingCustomCurrency")
model CustomCurrency {
  ...CurrencyBase;

  /**
   * Custom currency code based on Standard ISO 4217 code.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  code: CustomCurrencyCode;

  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  @visibility(Lifecycle.Read)
  created_at?: Shared.DateTime;
}
