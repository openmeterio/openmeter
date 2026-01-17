namespace Billing;

/**
 * Set of provider specific tax configs.
 */
@friendlyName("BillingTaxConfig")
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
  stripe?: TaxConfigStripe;

  /**
   * Custom invoicing tax config.
   */
  @summary("Custom invoicing tax config")
  custom_invoicing?: TaxConfigCustomInvoicing;
}

/**
 * Tax behavior.
 *
 * This enum is used to specify whether tax is included in the price or excluded from the price.
 */
@friendlyName("BillingTaxBehavior")
enum TaxBehavior {
  /**
   * Tax is included in the price.
   */
  Inclusive: "inclusive",

  /**
   * Tax is excluded from the price.
   */
  Exclusive: "exclusive",
}

/**
 * The tax config for Stripe.
 */
@friendlyName("BillingTaxConfigStripe")
model TaxConfigStripe {
  /**
   * Product [tax code](https://docs.stripe.com/tax/tax-codes).
   */
  @summary("Tax code")
  @pattern("^txcd_\\d{8}$")
  @example("txcd_10000000")
  code: string;
}

/**
 * Custom invoicing tax config.
 */
@friendlyName("BillingTaxConfigCustomInvoicing")
model TaxConfigCustomInvoicing {
  /**
   * The tax code should be interpreted by the custom invoicing provider.
   */
  @summary("Tax code")
  @maxLength(64)
  code: string;
}
