import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter;

/**
 * Alignment configuration for a plan or subscription.
 */
#deprecated "Not used anymore"
@friendlyName("Alignment")
model Alignment {
  /**
   * Whether all Billable items and RateCards must align.
   * Alignment means the Price's BillingCadence must align for both duration and anchor time.
   */
  #deprecated "Not used anymore"
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  billablesMustAlign?: boolean;
}
