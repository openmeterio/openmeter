import "@typespec/json-schema";
import "@openmeter/typespec-notification-events";
import "../../shared/index.tsp";

using TypeSpec.JsonSchema;
using NotificationCodegen;

/**
 * Notification Event contracts owned by the entitlements domain.
 */
namespace Entitlements.NotificationEvents;

/**
 * Condition kind of an entitlement balance threshold.
 */
@friendlyName("EntitlementBalanceThresholdType")
enum BalanceThresholdType {
  /**
   * Triggers when the remaining balance drops to or below the value.
   */
  BalanceAtOrBelow: "balance_at_or_below",

  /**
   * Triggers when the used percentage of the entitlement reaches the value.
   */
  UsagePercentageAtOrAbove: "usage_percentage_at_or_above",
}

/**
 * A single entitlement balance threshold.
 */
@friendlyName("EntitlementBalanceThreshold")
model BalanceThreshold {
  /**
   * Condition kind of the threshold.
   */
  type: BalanceThresholdType;

  /**
   * Threshold value, interpreted according to the threshold type.
   */
  value: float64;
}

/**
 * Rule configuration of `v1.entitlement.balance_threshold.triggered`.
 */
@jsonSchema
@id("v1.entitlement.balance_threshold.triggered.rule_config")
@friendlyName("EntitlementBalanceThresholdTriggeredV1RuleConfig")
model BalanceThresholdTriggeredV1RuleConfig {
  /**
   * Thresholds evaluated against the entitlement balance.
   */
  @minItems(1)
  thresholds: BalanceThreshold[];

  /**
   * Feature keys the Rule is limited to. All features when omitted.
   */
  features?: Shared.ResourceKey[];
}

/**
 * Payload of `v1.entitlement.balance_threshold.triggered`.
 */
@jsonSchema
@id("v1.entitlement.balance_threshold.triggered.payload")
@eventType(
  "v1.entitlement.balance_threshold.triggered",
  BalanceThresholdTriggeredV1RuleConfig
)
@friendlyName("EntitlementBalanceThresholdTriggeredV1Payload")
model BalanceThresholdTriggeredV1Payload {
  /**
   * Customer owning the entitlement.
   */
  customer_id: Shared.ULID;

  /**
   * Entitlement whose balance crossed the threshold.
   */
  entitlement_id: Shared.ULID;

  /**
   * Feature of the entitlement.
   */
  feature_key: Shared.ResourceKey;

  /**
   * Balance at evaluation time.
   */
  current_balance: float64;

  /**
   * Threshold that was crossed.
   */
  threshold: BalanceThreshold;
}
