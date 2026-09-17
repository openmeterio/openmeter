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
  @example(EntitlementType.Static)
  @visibility(Lifecycle.Read)
  type: EntitlementType;

  /**
   * The feature key of the entitlement.
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
   * Only available for metered entitlements. The current balance details of the
   * entitlement. Requires the `value` expand.
   */
  @visibility(Lifecycle.Read)
  value?: EntitlementAccessValue;
}

/**
 * Balance details of a metered entitlement.
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
   * The remaining balance of the entitlement in the current usage period.
   */
  @visibility(Lifecycle.Read)
  balance: Shared.Numeric;

  /**
   * The usage recorded in the current usage period.
   */
  @visibility(Lifecycle.Read)
  usage: Shared.Numeric;

  /**
   * The usage exceeding the available balance in the current usage period.
   */
  @visibility(Lifecycle.Read)
  overage: Shared.Numeric;

  /**
   * The total amount granted and currently available to the entitlement.
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
