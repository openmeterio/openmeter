import "../shared/index.tsp";
import "./event.tsp";

namespace Notifications;

/**
 * What a balance threshold is measured against.
 */
@friendlyName("NotificationBalanceThresholdType")
enum NotificationBalanceThresholdType {
  /**
   * The remaining balance in the current usage period.
   */
  BalanceValue: "balance_value",

  /**
   * The usage as a percentage of the total grants in the current usage period.
   */
  UsagePercentage: "usage_percentage",

  /**
   * The usage value in the current usage period.
   */
  UsageValue: "usage_value",
}

/**
 * A balance threshold of a notification rule. Crossing it generates an
 * `entitlements.balance.threshold` event.
 */
@friendlyName("NotificationBalanceThreshold")
model NotificationBalanceThreshold {
  /**
   * What the threshold value is measured against.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: NotificationBalanceThresholdType;

  /**
   * The threshold value.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  value: float64;
}

/**
 * The fields shared by every notification rule type.
 */
model NotificationRuleBase<T extends NotificationEventType> {
  ...OmitProperties<Shared.Resource, "description">;

  /**
   * The type of event the rule generates. Immutable after creation.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: T;

  /**
   * Whether the rule is disabled. Disabled rules do not generate events.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  disabled?: boolean = false;

  /**
   * The channels the rule delivers its events to. At least one and at most five
   * channels are required. Responses omit channels that have since been disabled or
   * deleted.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @maxItems(5)
  channel_ids: Array<Shared.ULID>;
}

/**
 * A rule that generates an event when an entitlement balance crosses one of its
 * thresholds.
 */
@friendlyName("NotificationRuleBalanceThreshold")
model NotificationRuleBalanceThreshold {
  ...NotificationRuleBase<NotificationEventType.EntitlementsBalanceThreshold>;

  /**
   * The thresholds that generate an event when crossed. Between 1 and 10 thresholds.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @minItems(1)
  @maxItems(10)
  thresholds: Array<NotificationBalanceThreshold>;

  /**
   * The features the rule applies to, by id or key. When omitted, the rule applies
   * to every feature.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  features?: Array<string>;
}

/**
 * A rule that generates an event when an entitlement usage period is reset.
 */
@friendlyName("NotificationRuleEntitlementReset")
model NotificationRuleEntitlementReset {
  ...NotificationRuleBase<NotificationEventType.EntitlementsReset>;

  /**
   * The features the rule applies to, by id or key. When omitted, the rule applies
   * to every feature.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  features?: Array<string>;
}

/**
 * A rule that generates an event when an invoice is created.
 */
@friendlyName("NotificationRuleInvoiceCreated")
model NotificationRuleInvoiceCreated {
  ...NotificationRuleBase<NotificationEventType.InvoiceCreated>;
}

/**
 * A rule that generates an event when an invoice is updated.
 */
@friendlyName("NotificationRuleInvoiceUpdated")
model NotificationRuleInvoiceUpdated {
  ...NotificationRuleBase<NotificationEventType.InvoiceUpdated>;
}

/**
 * A notification rule selects the type of event to generate, the conditions
 * specific to that type, and the channels to deliver the events to.
 */
@friendlyName("NotificationRule")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union NotificationRule {
  `entitlements.balance.threshold`: NotificationRuleBalanceThreshold,
  `entitlements.reset`: NotificationRuleEntitlementReset,
  `invoice.created`: NotificationRuleInvoiceCreated,
  `invoice.updated`: NotificationRuleInvoiceUpdated,
}

/**
 * Request body for a balance threshold rule.
 */
@withVisibility(Lifecycle.Create, Lifecycle.Update)
@friendlyName("NotificationRuleBalanceThresholdRequest")
model NotificationRuleBalanceThresholdRequest {
  ...NotificationRuleBalanceThreshold;
}

/**
 * Request body for an entitlement reset rule.
 */
@withVisibility(Lifecycle.Create, Lifecycle.Update)
@friendlyName("NotificationRuleEntitlementResetRequest")
model NotificationRuleEntitlementResetRequest {
  ...NotificationRuleEntitlementReset;
}

/**
 * Request body for an invoice created rule.
 */
@withVisibility(Lifecycle.Create, Lifecycle.Update)
@friendlyName("NotificationRuleInvoiceCreatedRequest")
model NotificationRuleInvoiceCreatedRequest {
  ...NotificationRuleInvoiceCreated;
}

/**
 * Request body for an invoice updated rule.
 */
@withVisibility(Lifecycle.Create, Lifecycle.Update)
@friendlyName("NotificationRuleInvoiceUpdatedRequest")
model NotificationRuleInvoiceUpdatedRequest {
  ...NotificationRuleInvoiceUpdated;
}

/**
 * Request body for creating or updating a notification rule. Updates replace the
 * rule's mutable state: omitting `disabled`, `labels`, or `features` resets them
 * to their defaults. The `type` must match the existing rule on update.
 */
@friendlyName("NotificationRuleRequest")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union NotificationRuleRequest {
  `entitlements.balance.threshold`: NotificationRuleBalanceThresholdRequest,
  `entitlements.reset`: NotificationRuleEntitlementResetRequest,
  `invoice.created`: NotificationRuleInvoiceCreatedRequest,
  `invoice.updated`: NotificationRuleInvoiceUpdatedRequest,
}
