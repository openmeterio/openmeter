namespace Entitlements;

/**
 * The type of the entitlement.
 */
@friendlyName("BillingEntitlementType")
enum EntitlementType {
  metered,
  static,
  boolean,
}

/**
 * Entitlement check result.
 */
@friendlyName("BillingEntitlementCheck")
model EntitlementCheck {
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
