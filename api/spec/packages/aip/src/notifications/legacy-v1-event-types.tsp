import "@typespec/json-schema";
import "@openmeter/typespec-notification-events";
import "../shared/index.tsp";
import "../features/index.tsp";
import "./legacy-v1.tsp";
import "./rule.tsp";

using TypeSpec.JsonSchema;
using NotificationCodegen;

/**
 * The legacy Event types as v1 Notification Event contracts. Each payload is the
 * `data` of the legacy payload as the v3 API returns it, and each Rule
 * configuration is the type-specific part of the v3 Rule. A payload's doc
 * comment is also its Event type's description in the API.
 */
namespace Notifications.LegacyV1;

/**
 * An invoice was created. The payload is the invoice as it was created.
 *
 * Legacy name: `invoice.created`.
 */
@jsonSchema
@id("v1.billing.invoice.created.payload")
@eventType("v1.billing.invoice.created", InvoiceCreatedV1RuleConfig)
@friendlyName("BillingInvoiceCreatedV1Payload")
model InvoiceCreatedV1Payload {
  ...V1Invoice;
}

/**
 * Rule configuration of `v1.billing.invoice.created`. Invoice Rules have no
 * type-specific settings.
 */
@jsonSchema
@id("v1.billing.invoice.created.rule_config")
@friendlyName("BillingInvoiceCreatedV1RuleConfig")
model InvoiceCreatedV1RuleConfig {}

/**
 * An invoice was updated. The payload is the invoice after the update.
 *
 * Legacy name: `invoice.updated`.
 */
@jsonSchema
@id("v1.billing.invoice.updated.payload")
@eventType("v1.billing.invoice.updated", InvoiceUpdatedV1RuleConfig)
@friendlyName("BillingInvoiceUpdatedV1Payload")
model InvoiceUpdatedV1Payload {
  ...V1Invoice;
}

/**
 * Rule configuration of `v1.billing.invoice.updated`. Invoice Rules have no
 * type-specific settings.
 */
@jsonSchema
@id("v1.billing.invoice.updated.rule_config")
@friendlyName("BillingInvoiceUpdatedV1RuleConfig")
model InvoiceUpdatedV1RuleConfig {}

/**
 * A metered entitlement's balance crossed one of the thresholds configured on the
 * Rule. The payload carries the entitlement, its feature, subject, customer and
 * value, and the threshold that was crossed.
 *
 * Legacy name: `entitlements.balance.threshold`.
 */
@jsonSchema
@id("v1.entitlement.balance_threshold.triggered.payload")
@eventType(
  "v1.entitlement.balance_threshold.triggered",
  BalanceThresholdTriggeredV1RuleConfig
)
@friendlyName("EntitlementBalanceThresholdTriggeredV1Payload")
model BalanceThresholdTriggeredV1Payload {
  ...V1NotificationEventBalanceThresholdPayloadData;
}

/**
 * Rule configuration of `v1.entitlement.balance_threshold.triggered`.
 */
@jsonSchema
@id("v1.entitlement.balance_threshold.triggered.rule_config")
@friendlyName("EntitlementBalanceThresholdTriggeredV1RuleConfig")
model BalanceThresholdTriggeredV1RuleConfig {
  /**
   * The thresholds that generate an event when crossed.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @minItems(1)
  @maxItems(10)
  thresholds: Array<NotificationBalanceThreshold>;

  ...EntitlementFeaturesRuleConfig;
}

/**
 * A metered entitlement's usage period was reset. The payload carries the
 * entitlement, its feature, subject, customer and value.
 *
 * Legacy name: `entitlements.reset`.
 */
@jsonSchema
@id("v1.entitlement.usage_period.reset.payload")
@eventType("v1.entitlement.usage_period.reset", UsagePeriodResetV1RuleConfig)
@friendlyName("EntitlementUsagePeriodResetV1Payload")
model UsagePeriodResetV1Payload {
  ...V1NotificationEventEntitlementValuePayloadBase;
}

/**
 * Rule configuration of `v1.entitlement.usage_period.reset`.
 */
@jsonSchema
@id("v1.entitlement.usage_period.reset.rule_config")
@friendlyName("EntitlementUsagePeriodResetV1RuleConfig")
model UsagePeriodResetV1RuleConfig {
  ...EntitlementFeaturesRuleConfig;
}

/**
 * The feature filter shared by the entitlement Rule configurations.
 */
@friendlyName("EntitlementFeaturesRuleConfig")
model EntitlementFeaturesRuleConfig {
  /**
   * The features the rule applies to. When omitted, the rule applies to every
   * feature.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  features?: Array<Shared.ResourceReference<Features.Feature>>;
}
