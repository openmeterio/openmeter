namespace OpenMeter;

/**
 * Pro-rating mode options for handling billing period changes.
 */
@friendlyName("ProRatingMode")
union ProRatingMode {
  /**
   * Calculate pro-rated charges based on time remaining in billing period.
   */
  proratePrices: "prorate_prices",
}

/**
 * Configuration for pro-rating behavior.
 */
@friendlyName("ProRatingConfig")
model ProRatingConfig {
  /**
   * Whether pro-rating is enabled for this plan.
   */
  @summary("Enable pro-rating")
  enabled: boolean = true;

  /**
   * How to handle pro-rating for billing period changes.
   */
  @summary("Pro-rating mode")
  mode: ProRatingMode = ProRatingMode.proratePrices;
}
