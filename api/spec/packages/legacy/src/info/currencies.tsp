namespace OpenMeter;

/**
 * Currency describes a currency supported by OpenMeter.
 */
@friendlyName("Currency")
model Currency {
  /**
   * The currency ISO code.
   */
  code: CurrencyCode;

  /**
   * The currency name.
   */
  name: string;

  /**
   * The currency symbol.
   */
  symbol: string;

  /**
   * Subunit of the currency.
   */
  subunits: uint32;
}
