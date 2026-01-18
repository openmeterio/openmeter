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
   * External invoicing tax config.
   */
  @summary("External invoicing tax config")
  external_invoicing?: TaxConfigExternalInvoicing;
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
 * External invoicing tax config.
 */
@friendlyName("BillingTaxConfigExternalInvoicing")
model TaxConfigExternalInvoicing {
  /**
   * The tax code should be interpreted by the external invoicing provider.
   */
  @summary("Tax code")
  @maxLength(64)
  code: string;
}
