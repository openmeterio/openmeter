namespace OpenMeter;

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

  /**
   * Custom invoicing tax config.
   */
  @summary("Custom invoicing tax config")
  customInvoicing?: CustomInvoicingTaxConfig;
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

/**
 * Custom invoicing tax config.
 */
@friendlyName("CustomInvoicingTaxConfig")
model CustomInvoicingTaxConfig {
  /**
   * Tax code.
   *
   * The tax code should be interpreted by the custom invoicing provider.
   */
  @summary("Tax code")
  code: string;
}
