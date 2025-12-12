import "./entitlements.tsp";

namespace Entitlements;

/**
 * Entitlement access result.
 */
@friendlyName("BillingEntitlementAccessResult")
model EntitlementAccessResult {
  /**
   * The type of the entitlement.
   */
  @visibility(Lifecycle.Read)
  type: EntitlementType;

  /**
   * Whether the customer has access to the feature.
   */
  @visibility(Lifecycle.Read)
  has_access: boolean;

  /**
   * Only available for static entitlements.
   */
  @example("{ \"rateLimit\": 100 }")
  @encode("json")
  @visibility(Lifecycle.Read)
  config?: string;
}
