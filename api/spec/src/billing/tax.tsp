namespace OpenMeter.Billing;

/**
 * A tax config.
 */
@friendlyName("TaxConfig")
model TaxConfig {
  /**
   * Stripe tax config.
   */
  @summary("Stripe tax config")
  stripe?: {
    @summary("Tax code")
    @pattern("^txcd_\\d{8}$")
    @example("txcd_10000000")
    code: string;
  };
}
