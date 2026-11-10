import "@typespec/json-schema";
import "@openmeter/typespec-notification-events";
import "../../shared/index.tsp";
import "../../customers/customer.tsp";
import "../../customers/entitlements/entitlement.tsp";
import "../../entitlements/access.tsp";
import "../../features/feature.tsp";

using TypeSpec.JsonSchema;
using NotificationCodegen;

namespace Entitlements.NotificationEvents;

/**
 * Rule configuration of `v1.entitlement.usage.reset`.
 */
@jsonSchema
@id("v1.entitlement.usage.reset.rule_config")
@friendlyName("EntitlementUsageResetV1RuleConfig")
model UsageResetV1RuleConfig {
  /**
   * Feature keys the Rule is limited to. All features when omitted.
   */
  features?: Shared.ResourceKey[];
}

/**
 * Payload of `v1.entitlement.usage.reset`.
 *
 * The entitlement, feature, customer and value have the shapes the v3 API
 * returns. A change to those models changes this contract, so definitions.sum
 * fails until the change is either accepted as a new Event type version or
 * this payload is frozen into dedicated models.
 */
@jsonSchema
@id("v1.entitlement.usage.reset.payload")
@eventType("v1.entitlement.usage.reset", UsageResetV1RuleConfig)
@friendlyName("EntitlementUsageResetV1Payload")
model UsageResetV1Payload {
  /**
   * Entitlement whose usage was reset.
   */
  entitlement: Customers.EntitlementMetered;

  /**
   * Feature of the entitlement.
   */
  feature: Features.Feature;

  /**
   * Customer owning the entitlement.
   */
  customer: Customers.Customer;

  /**
   * Value of the entitlement right after the reset.
   */
  value: Entitlements.EntitlementValueResult;
}
