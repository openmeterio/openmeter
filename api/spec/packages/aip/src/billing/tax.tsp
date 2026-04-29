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
   *
   * @deprecated Use `tax_code_id` instead. When both `stripe.code` and `tax_code_id` are provided,
   * `tax_code_id` takes precedence and `stripe.code` is ignored.
   */
  #deprecated "Use tax_code_id instead"
  @summary("Stripe tax config")
  stripe?: TaxConfigStripe;

  /**
   * External invoicing tax config.
   *
   * @deprecated Use `tax_code_id` instead.
   */
  #deprecated "Use tax_code_id instead"
  @summary("External invoicing tax config")
  external_invoicing?: TaxConfigExternalInvoicing;

  /**
   * Tax code reference.
   *
   * When both `tax_code_id` and `stripe.code` are provided, `tax_code_id` takes precedence:
   * the referenced tax code entity is used and `stripe.code` is ignored.
   */
  @summary("Tax code ID")
  tax_code_id?: Shared.ULID;
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
