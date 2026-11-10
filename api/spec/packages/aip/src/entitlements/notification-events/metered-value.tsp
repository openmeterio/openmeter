import "../../shared/index.tsp";
import "../../entitlements/access.tsp";

namespace Entitlements.NotificationEvents;

/**
 * Value of a metered entitlement when the Event happened.
 *
 * The v3 value result with its balance details required. The API returns them
 * only on request, but an Event payload never changes after it is created, so
 * a missing balance could not be recovered later.
 */
@friendlyName("EntitlementMeteredValueResult")
model MeteredValueResult {
  ...OmitProperties<Entitlements.EntitlementValueResult, "value">;

  /**
   * The balance details of the entitlement when the Event happened.
   */
  value: Entitlements.EntitlementAccessValue;
}
