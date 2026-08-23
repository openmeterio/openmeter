import "../shared/index.tsp";

namespace Notifications;

/**
 * The type of a notification event. The event type determines which payload
 * variant the event carries and which rule configuration produced it.
 */
@friendlyName("NotificationEventType")
enum NotificationEventType {
  /**
   * A metered entitlement's balance crossed one of the thresholds configured on the
   * rule.
   */
  EntitlementsBalanceThreshold: "entitlements_balance_threshold",

  /**
   * A metered entitlement's usage period was reset.
   */
  EntitlementsReset: "entitlements_reset",

  /**
   * An invoice was created.
   */
  InvoiceCreated: "invoice_created",

  /**
   * An invoice was updated.
   */
  InvoiceUpdated: "invoice_updated",
}

/**
 * The delivery state of a notification event towards a single channel.
 */
@friendlyName("NotificationEventDeliveryState")
enum NotificationEventDeliveryState {
  /**
   * The event was delivered and the recipient acknowledged it.
   */
  Success: "success",

  /**
   * Delivery failed and no further attempts will be made.
   */
  Failed: "failed",

  /**
   * Delivery is in progress.
   */
  Sending: "sending",

  /**
   * Delivery has not been attempted yet.
   */
  Pending: "pending",

  /**
   * Delivery is being retried after an explicit resend request.
   */
  Resending: "resending",
}

/**
 * A reference to the notification rule that generated an event. Notification rules
 * are not yet exposed as a v3 resource, so events carry an inline reference rather
 * than a link to a retrievable resource.
 */
@friendlyName("NotificationRuleReference")
model NotificationRuleReference {
  /**
   * The unique identifier of the rule.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of event the rule generates.
   */
  @visibility(Lifecycle.Read)
  type: NotificationEventType;

  /**
   * The user provided name of the rule.
   */
  @visibility(Lifecycle.Read)
  name: string;
}

/**
 * The response the recipient returned for a delivery attempt. For webhook channels
 * this is the HTTP response; fields that the transport could not observe are
 * omitted.
 */
@friendlyName("NotificationEventDeliveryAttemptResponse")
model NotificationEventDeliveryAttemptResponse {
  /**
   * The HTTP status code returned by the recipient, if a response was received at
   * all.
   */
  @visibility(Lifecycle.Read)
  status_code?: int32;

  /**
   * The response body returned by the recipient. Empty when the recipient returned
   * no body or the request never completed.
   */
  @visibility(Lifecycle.Read)
  body: string;

  /**
   * How long the delivery attempt took, in milliseconds.
   */
  @visibility(Lifecycle.Read)
  duration_ms: int64;

  /**
   * The URL the event was delivered to, for webhook channels.
   */
  @visibility(Lifecycle.Read)
  url?: string;
}

/**
 * A single delivery attempt towards a channel.
 */
@friendlyName("NotificationEventDeliveryAttempt")
model NotificationEventDeliveryAttempt {
  /**
   * The state the delivery reached with this attempt.
   */
  @visibility(Lifecycle.Read)
  state: NotificationEventDeliveryState;

  /**
   * The response returned by the recipient.
   */
  @visibility(Lifecycle.Read)
  response: NotificationEventDeliveryAttemptResponse;

  /**
   * When the attempt was made.
   */
  @visibility(Lifecycle.Read)
  timestamp: Shared.DateTime;
}

/**
 * The delivery status of a notification event towards one of the channels the
 * generating rule targets. An event has one entry per channel.
 */
@friendlyName("NotificationEventDeliveryStatus")
model NotificationEventDeliveryStatus {
  /**
   * The identifier of the channel this status belongs to. Retrieve the channel
   * itself through the notification channel endpoints.
   */
  @visibility(Lifecycle.Read)
  channel_id: Shared.ULID;

  /**
   * The current delivery state.
   */
  @visibility(Lifecycle.Read)
  state: NotificationEventDeliveryState;

  /**
   * The reason for the last state change. Empty when the last change needs no
   * explanation, for example a successful delivery.
   */
  @visibility(Lifecycle.Read)
  reason: string;

  /**
   * When the delivery state was last updated.
   */
  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;

  /**
   * When the next delivery attempt is scheduled. Absent when no further attempts
   * will be made, either because delivery succeeded or because it failed
   * permanently.
   */
  @visibility(Lifecycle.Read)
  next_attempt?: Shared.DateTime;

  /**
   * The delivery attempts made so far, most recent first.
   */
  @visibility(Lifecycle.Read)
  attempts: Array<NotificationEventDeliveryAttempt>;
}

/**
 * The entitlement balance at the time the event was generated.
 */
@friendlyName("NotificationEventEntitlementValue")
model NotificationEventEntitlementValue {
  /**
   * Whether the subject had access to the feature. Balance never turns negative, so
   * access can be lost while the balance is still reported as zero.
   */
  @visibility(Lifecycle.Read)
  has_access: boolean;

  /**
   * The remaining balance of the entitlement.
   */
  @visibility(Lifecycle.Read)
  balance?: float64;

  /**
   * The total feature usage in the current usage period.
   */
  @visibility(Lifecycle.Read)
  usage?: float64;

  /**
   * The usage that was not covered by any grant.
   */
  @visibility(Lifecycle.Read)
  overage?: float64;
}

/**
 * A reference to the feature an entitlement notification event refers to.
 */
@friendlyName("NotificationEventFeatureReference")
model NotificationEventFeatureReference {
  /**
   * The unique identifier of the feature.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The immutable key of the feature.
   */
  @visibility(Lifecycle.Read)
  key: string;
}

/**
 * The entities an entitlement notification event refers to. Each is identified
 * rather than embedded; retrieve the full resources through their own endpoints.
 */
@friendlyName("NotificationEventEntitlementData")
model NotificationEventEntitlementData {
  /**
   * The identifier of the entitlement whose balance triggered the event.
   */
  @visibility(Lifecycle.Read)
  entitlement_id: Shared.ULID;

  /**
   * The feature the entitlement grants access to.
   */
  @visibility(Lifecycle.Read)
  feature: NotificationEventFeatureReference;

  /**
   * The key of the subject the entitlement belongs to.
   */
  @visibility(Lifecycle.Read)
  subject_key: string;

  /**
   * The identifier of the customer the subject is attributed to, when the subject is
   * attributed to one.
   */
  @visibility(Lifecycle.Read)
  customer_id?: Shared.ULID;

  /**
   * The entitlement balance at the time the event was generated.
   */
  @visibility(Lifecycle.Read)
  value: NotificationEventEntitlementValue;
}

/**
 * The kind of threshold that a balance threshold rule triggers on.
 *
 * The legacy `NUMBER` and `PERCENT` values are normalized to `usage_value` and
 * `usage_percentage` respectively, so events created before the rename report the
 * current value.
 */
@friendlyName("NotificationEventBalanceThresholdType")
enum NotificationEventBalanceThresholdType {
  /**
   * The remaining balance, based on usage and the total of grants in the current usage
   * period.
   */
  BalanceValue: "balance_value",

  /**
   * The usage as a percentage of the total of grants in the current usage period.
   */
  UsagePercentage: "usage_percentage",

  /**
   * The usage value in the current usage period.
   */
  UsageValue: "usage_value",
}

/**
 * The threshold that the entitlement balance crossed.
 */
@friendlyName("NotificationEventBalanceThreshold")
model NotificationEventBalanceThreshold {
  /**
   * What the threshold value is measured against.
   */
  @visibility(Lifecycle.Read)
  type: NotificationEventBalanceThresholdType;

  /**
   * The threshold value that was crossed.
   */
  @visibility(Lifecycle.Read)
  value: float64;
}

/**
 * The entities and threshold a balance threshold event refers to.
 */
@friendlyName("NotificationEventBalanceThresholdData")
model NotificationEventBalanceThresholdData {
  ...NotificationEventEntitlementData;

  /**
   * The threshold the balance crossed.
   */
  @visibility(Lifecycle.Read)
  threshold: NotificationEventBalanceThreshold;
}

/**
 * A reference to the invoice an invoice notification event refers to.
 */
@friendlyName("NotificationEventInvoiceReference")
model NotificationEventInvoiceReference {
  /**
   * The unique identifier of the invoice.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The human readable invoice number.
   */
  @visibility(Lifecycle.Read)
  number: string;
}

/**
 * The invoice an invoice notification event refers to. The invoice itself is
 * identified rather than embedded; retrieve it through the invoice endpoints.
 */
@friendlyName("NotificationEventInvoiceData")
model NotificationEventInvoiceData {
  /**
   * The invoice the event was generated for.
   */
  @visibility(Lifecycle.Read)
  invoice: NotificationEventInvoiceReference;

  /**
   * The identifier of the customer the invoice was issued to.
   */
  @visibility(Lifecycle.Read)
  customer_id?: Shared.ULID;

  /**
   * The currency the invoice is denominated in.
   */
  @visibility(Lifecycle.Read)
  currency: Shared.CurrencyCode;

  /**
   * The status of the invoice at the time the event was generated.
   */
  @visibility(Lifecycle.Read)
  status: string;

  /**
   * The total value of the invoice, including taxes, at the time the event was
   * generated.
   */
  @visibility(Lifecycle.Read)
  total: Shared.Numeric;
}

/**
 * A balance threshold notification event payload.
 */
@friendlyName("NotificationEventBalanceThresholdPayload")
model NotificationEventBalanceThresholdPayload {
  /**
   * The identifier of the event the payload belongs to.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of the event.
   */
  @visibility(Lifecycle.Read)
  type: NotificationEventType.EntitlementsBalanceThreshold;

  /**
   * When the event was generated.
   */
  @visibility(Lifecycle.Read)
  timestamp: Shared.DateTime;

  /**
   * The entities and threshold the event refers to.
   */
  @visibility(Lifecycle.Read)
  data: NotificationEventBalanceThresholdData;
}

/**
 * An entitlement reset notification event payload.
 */
@friendlyName("NotificationEventResetPayload")
model NotificationEventResetPayload {
  /**
   * The identifier of the event the payload belongs to.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of the event.
   */
  @visibility(Lifecycle.Read)
  type: NotificationEventType.EntitlementsReset;

  /**
   * When the event was generated.
   */
  @visibility(Lifecycle.Read)
  timestamp: Shared.DateTime;

  /**
   * The entities the event refers to.
   */
  @visibility(Lifecycle.Read)
  data: NotificationEventEntitlementData;
}

/**
 * An invoice created notification event payload.
 */
@friendlyName("NotificationEventInvoiceCreatedPayload")
model NotificationEventInvoiceCreatedPayload {
  /**
   * The identifier of the event the payload belongs to.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of the event.
   */
  @visibility(Lifecycle.Read)
  type: NotificationEventType.InvoiceCreated;

  /**
   * When the event was generated.
   */
  @visibility(Lifecycle.Read)
  timestamp: Shared.DateTime;

  /**
   * The invoice the event refers to.
   */
  @visibility(Lifecycle.Read)
  data: NotificationEventInvoiceData;
}

/**
 * An invoice updated notification event payload.
 */
@friendlyName("NotificationEventInvoiceUpdatedPayload")
model NotificationEventInvoiceUpdatedPayload {
  /**
   * The identifier of the event the payload belongs to.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of the event.
   */
  @visibility(Lifecycle.Read)
  type: NotificationEventType.InvoiceUpdated;

  /**
   * When the event was generated.
   */
  @visibility(Lifecycle.Read)
  timestamp: Shared.DateTime;

  /**
   * The invoice the event refers to.
   */
  @visibility(Lifecycle.Read)
  data: NotificationEventInvoiceData;
}

/**
 * The payload delivered to the channels of the generating rule, discriminated by
 * the event type.
 */
@friendlyName("NotificationEventPayload")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union NotificationEventPayload {
  entitlements_balance_threshold: NotificationEventBalanceThresholdPayload,
  entitlements_reset: NotificationEventResetPayload,
  invoice_created: NotificationEventInvoiceCreatedPayload,
  invoice_updated: NotificationEventInvoiceUpdatedPayload,
}

/**
 * A notification event records that a notification rule fired, and tracks the
 * delivery of the resulting payload to each of the rule's channels. Events are
 * created by the system and cannot be modified.
 */
@friendlyName("NotificationEvent")
model NotificationEvent {
  /**
   * The unique identifier of the event.
   */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /**
   * The type of the event.
   */
  @visibility(Lifecycle.Read)
  type: NotificationEventType;

  /**
   * When the event was generated.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * The rule that generated the event.
   */
  @visibility(Lifecycle.Read)
  rule: NotificationRuleReference;

  /**
   * The delivery status of the event, one entry per channel the generating rule
   * targets.
   */
  @visibility(Lifecycle.Read)
  delivery_status: Array<NotificationEventDeliveryStatus>;

  /**
   * The payload delivered to the channels.
   */
  @visibility(Lifecycle.Read)
  payload: NotificationEventPayload;
}

/**
 * Request body for re-sending a notification event.
 */
@friendlyName("ResendNotificationEventRequest")
model NotificationEventResendRequest {
  /**
   * The channels to re-send the event to. When omitted or empty the event is re-sent
   * to every channel of the generating rule that is eligible for a resend. Channels
   * that are unknown to the rule or disabled are rejected.
   */
  @visibility(Lifecycle.Create)
  channels?: Array<Shared.ULID>;
}
