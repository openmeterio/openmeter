import "../shared/index.tsp";

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
