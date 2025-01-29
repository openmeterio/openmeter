import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * Alignment configuration for a plan or subscription.
 */
@friendlyName("Alignment")
model Alignment {
  /**
   * Whether all Billable items and RateCards must align.
   * Alignment means the Price's BillingCadence must align for both duration and anchor time.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  billablesMustAlign?: boolean;
}
