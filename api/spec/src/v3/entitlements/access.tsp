import "./entitlements.tsp";
import "../shared/index.tsp";

namespace Entitlements;

/**
 * Entitlement access result.
 */
@friendlyName("BillingEntitlementAccessResult")
model EntitlementAccessResult {
  /**
   * The type of the entitlement.
   */
  @example(EntitlementType.static)
  @visibility(Lifecycle.Read)
  type: EntitlementType;

  /**
   * The feature key of the entitlement.
   */
  @example("available_models")
  @visibility(Lifecycle.Read)
  feature_key: Shared.ResourceKey;

  /**
   * Whether the customer has access to the feature.
   * Always true for `boolean` and `static` entitlements.
   * Depends on balance for `metered` entitlements.
   */
  @example(true)
  @visibility(Lifecycle.Read)
  has_access: boolean;

  /**
   * Only available for static entitlements.
   * Config is the JSON parsable configuration of the entitlement.
   * Useful to describe per customer configuration.
   */
  @example("{ \"availableModels\": [\"gpt-5\", \"gpt-4o\"] }")
  @visibility(Lifecycle.Read)
  config?: string;
}
