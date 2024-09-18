namespace OpenMeter;

/**
 * A billing provider.
 */
@friendlyName("BillingProvider")
enum BillingProvider {
  stripe: "stripe",
}

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
