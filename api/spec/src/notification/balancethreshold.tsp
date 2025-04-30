import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Notification;

/**
 * Payload for notification event with `entitlements.balance.threshold` type.
 */
@friendlyName("NotificationEventBalanceThresholdPayload")
model NotificationEventBalanceThresholdPayload {
  /**
   * A unique identifier for the notification event the payload belongs to.
   */
  @visibility(Lifecycle.Read)
  @summary("Notification Event Identifier")
  @example("01J2KNP1YTXQRXHTDJ4KPR7PZ0")
  id: ULID;

  /**
   * Type of the notification event.
   */
  @visibility(Lifecycle.Read)
  @summary("Notification Event Type")
  type: NotificationEventType.entitlementsBalanceThreshold;

  /**
   * Timestamp when the notification event was created in RFC 3339 format.
   */
  @visibility(Lifecycle.Read)
  @summary("Creation Time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  timestamp: DateTime;

  /**
   * The data of the payload.
   */
  @visibility(Lifecycle.Read)
  @summary("Payload Data")
  data: NotificationEventBalanceThresholdPayloadData;
}

/**
 * Data of the payload for notification event with `entitlements.balance.threshold` type.
 */
@friendlyName("NotificationEventBalanceThresholdPayloadData")
model NotificationEventBalanceThresholdPayloadData {
  @visibility(Lifecycle.Read)
  @summary("Entitlement")
  entitlement: OpenMeter.Entitlements.EntitlementMetered;

  @visibility(Lifecycle.Read)
  @summary("Feature")
  feature: OpenMeter.Entitlements.Feature;

  @visibility(Lifecycle.Read)
  @summary("Subject")
  subject: OpenMeter.Subject;

  @visibility(Lifecycle.Read)
  @summary("Entitlement Value")
  value: OpenMeter.Entitlements.EntitlementValue;

  @visibility(Lifecycle.Read)
  @summary("Threshold")
  threshold: NotificationRuleBalanceThresholdValue;
}

/**
 * Threshold value with multiple supported types.
 */
@friendlyName("NotificationRuleBalanceThresholdValue")
model NotificationRuleBalanceThresholdValue {
  /**
   * Value of the threshold.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Threshold Value")
  @example(100)
  value: float64;

  /**
   * Type of the threshold.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @example(NotificationRuleBalanceThresholdValueType.NUMBER)
  type: NotificationRuleBalanceThresholdValueType;
}

/**
 * Type of the rule in the balance threshold specification.
 */
@summary("Notification balance threshold type")
@friendlyName("NotificationRuleBalanceThresholdValueType")
@extension("x-enum-varnames", #["Percent", "Number"])
enum NotificationRuleBalanceThresholdValueType {
  PERCENT,
  NUMBER,
}

/**
 * Limited representation of a feature resource which includes only its unique identifiers (id, key).
 */
@friendlyName("FeatureMeta")
model FeatureMeta {
  /**
   * Unique identifier of a feature.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Feature Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * The key is an immutable unique identifier of the feature used throughout the API,
   * for example when interacting with a subject's entitlements.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Feature Key")
  @example("gpt4_tokens")
  key: string;
}

/**
 * Notification rule with entitlements.balance.threshold type.
 */
@friendlyName("NotificationRuleBalanceThreshold")
model NotificationRuleBalanceThreshold {
  ...NotificationRuleCommon<NotificationEventType.entitlementsBalanceThreshold>;

  /**
   * List of thresholds the rule suppose to be triggered.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Entitlement Balance Thresholds")
  @minItems(1)
  @maxItems(10)
  thresholds: Array<NotificationRuleBalanceThresholdValue>;

  /**
   * Optional field containing list of features the rule applies to.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Features")
  @minItems(1)
  features?: Array<FeatureMeta>;
}

/**
 * Request with input parameters for creating new notification rule with entitlements.balance.threashold type.
 */
@friendlyName("NotificationRuleBalanceThresholdCreateRequest")
@withVisibility(Lifecycle.Create, Lifecycle.Update)
model NotificationRuleBalanceThresholdCreateRequest {
  ...OmitProperties<NotificationRuleBalanceThreshold, "channels" | "features">;

  /**
   * List of notification channels the rule is applied to.
   */
  @visibility(Lifecycle.Create, Lifecycle.Update)
  @summary("Channels")
  @minItems(1)
  channels: Array<ULID>;

  /**
   * Optional field for defining the scope of notification by feature. It may contain features by id or key.
   */
  @visibility(Lifecycle.Create, Lifecycle.Update)
  @summary("Features")
  @minItems(1)
  features?: Array<ULIDOrKey>;
}
