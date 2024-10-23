namespace OpenMeter.ProductCatalog;

/**
 * Set of provider specific tax configs.
 */
@friendlyName("TaxConfig")
model TaxConfig {
  /**
   * Stripe tax config.
   */
  @summary("Stripe tax config")
  stripe?: StripeTaxConfig;
}

/**
 * The tax config for Stripe.
 */
@friendlyName("StripeTaxConfig")
model StripeTaxConfig {
  /**
   * Product tax code.
   *
   * See: https://docs.stripe.com/tax/tax-codes
   */
  @summary("Tax code")
  @pattern("^txcd_\\d{8}$")
  @example("txcd_10000000")
  code: string;
}
