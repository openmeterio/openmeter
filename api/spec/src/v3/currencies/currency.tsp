import "../shared/index.tsp";

namespace Currencies;

/**
 * Currency describes a currency supported by the billing system.
 */
@friendlyName("BillingCurrency")
model Currency {
  /**
   * Standard ISO 4217 code.
   */
  @visibility(Lifecycle.Read)
  code: Shared.CurrencyCode;

  /**
   * English name of the currency.
   */
  @visibility(Lifecycle.Read)
  name: string;

  /**
   * Symbol representation of the currency.
   */
  @visibility(Lifecycle.Read)
  symbol: string;

  /**
   * Disambiguation symbol for the currency.
   */
  @visibility(Lifecycle.Read)
  disambiguate_symbol: string;

  /**
   * Number of subunits in the currency.
   */
  @visibility(Lifecycle.Read)
  subunits: uint32;

  /**
   * Smallest denomination of the currency.
   */
  @visibility(Lifecycle.Read)
  smallest_denomination: uint8;
}
