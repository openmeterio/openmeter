import "./entitlements.tsp";
import "../shared/index.tsp";

namespace Entitlements;

/**
 * Entitlement value result.
 */
@friendlyName("BillingEntitlementValueResult")
model EntitlementValueResult {
  /**
   * The type of the entitlement.
   */
  @example(EntitlementType.Static)
  @visibility(Lifecycle.Read)
  type: EntitlementType;

  /**
   * The feature key being evaluated.
   */
  @example("available_models")
  @visibility(Lifecycle.Read)
  feature_key: Shared.ResourceKey;

  /**
   * Whether the customer has access to the feature. Always true for `boolean` and
   * `static` entitlements. Depends on balance for `metered` entitlements.
   */
  @example(true)
  @visibility(Lifecycle.Read)
  has_access: boolean;

  /**
   * Only available for static entitlements. Config is the JSON parsable
   * configuration of the entitlement. Useful to describe per customer configuration.
   */
  @example("{ \"availableModels\": [\"gpt-5\", \"gpt-4o\"] }")
  @visibility(Lifecycle.Read)
  config?: string;

  /**
   * Only available for metered entitlements. The balance details of the entitlement
   * at the evaluation time. Requires the `value` expand.
   */
  @visibility(Lifecycle.Read)
  value?: EntitlementAccessValue;
}

/**
 * Entitlement access check result.
 */
@friendlyName("BillingEntitlementAccessCheckResult")
model EntitlementAccessCheckResult {
  ...PickProperties<EntitlementValueResult, "has_access" | "config">;

  /**
   * The type of the entitlement.
   *
   * If not provided, the feature has no entitlement defined (has access is always
   * false in this case)
   */
  @visibility(Lifecycle.Read)
  type?: EntitlementType;
}

/**
 * Entitlement value looked up by feature key. A missing entitlement has no type
 * and does not grant access.
 */
@friendlyName("BillingEntitlementFeatureValueResult")
model EntitlementFeatureValueResult {
  ...OmitProperties<EntitlementValueResult, "type">;

  /**
   * The type of the entitlement.
   *
   * If not provided, the feature has no entitlement defined (has access is always
   * false in this case)
   */
  @visibility(Lifecycle.Read)
  type?: EntitlementType;
}

/**
 * Balance details of a metered entitlement at the evaluation time, which is the
 * `at` query parameter when given and the current time otherwise.
 */
@friendlyName("BillingEntitlementAccessValue")
@example(#{
  balance: "100",
  usage: "50",
  overage: "0",
  total_available_grant_amount: "100",
  grant_balances: #{ additional_property: "1" },
})
model EntitlementAccessValue {
  /**
   * The remaining balance of the entitlement in the usage period at the evaluation
   * time.
   */
  @visibility(Lifecycle.Read)
  balance: Shared.Numeric;

  /**
   * The usage recorded in the usage period at the evaluation time.
   */
  @visibility(Lifecycle.Read)
  usage: Shared.Numeric;

  /**
   * The usage exceeding the available balance in the usage period at the evaluation
   * time.
   */
  @visibility(Lifecycle.Read)
  overage: Shared.Numeric;

  /**
   * The total amount granted and available to the entitlement at the evaluation
   * time.
   */
  @visibility(Lifecycle.Read)
  total_available_grant_amount: Shared.Numeric;

  /**
   * The remaining balance of each grant, keyed by grant ID.
   */
  @visibility(Lifecycle.Read)
  grant_balances: Record<Shared.Numeric>;
}

/**
 * Expands for customer entitlement access.
 *
 * Values:
 *
 * - `value`: The balance details of a metered entitlement; it sets the `value`
 *   field.
 */
@friendlyName("BillingEntitlementAccessExpand")
@summary("Customer entitlement access expands")
enum EntitlementAccessExpand {
  Value: "value",
}
