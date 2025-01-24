namespace OpenMeter.ProductCatalog;

/**
 * Set of provider specific tax configs.
 */
@friendlyName("TaxConfig")
model TaxConfig {
  /**
   * Tax behavior.
   *
   * If not specified the billing profile is used to determine the tax behavior.
   * If not specified in the billing profile, the provider's default behavior is used.
   */
  @summary("Tax behavior")
  behavior?: TaxBehavior;

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

/**
 * Tax behavior.
 *
 * This enum is used to specify whether tax is included in the price or excluded from the price.
 */
@friendlyName("TaxBehavior")
enum TaxBehavior {
  /**
   * Tax is included in the price.
   */
  inclusive: "inclusive",

  /**
   * Tax is excluded from the price.
   */
  exclusive: "exclusive",
}
