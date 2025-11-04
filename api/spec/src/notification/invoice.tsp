import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Payload for notification event with `invoice.*` type.
 */
@friendlyName("NotificationEventInvoicePayload")
model NotificationEventInvoicePayload<T extends NotificationEventType.invoiceCreated | NotificationEventType.invoiceUpdated> {
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
  type: T;

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
  data: Invoice;
}

/**
 * Payload for notification event with `invoice.created` type.
 */
@friendlyName("NotificationEventInvoiceCreatedPayload")
model NotificationEventInvoiceCreatedPayload {
  ...NotificationEventInvoicePayload<NotificationEventType.invoiceCreated>;
}

/**
 * Payload for notification event with `invoice.updated` type.
 */
@friendlyName("NotificationEventInvoiceUpdatedPayload")
model NotificationEventInvoiceUpdatedPayload {
  ...NotificationEventInvoicePayload<NotificationEventType.invoiceUpdated>;
}

/**
 * Notification rule with invoice.created type.
 */
@friendlyName("NotificationRuleInvoiceCreated")
model NotificationRuleInvoiceCreated {
  ...NotificationRuleCommon<NotificationEventType.invoiceCreated>;
}

/**
 * Notification rule with invoice.updated type.
 */
@friendlyName("NotificationRuleInvoiceUpdated")
model NotificationRuleInvoiceUpdated {
  ...NotificationRuleCommon<NotificationEventType.invoiceUpdated>;
}

/**
 * Request with input parameters for creating new notification rule with invoice.created type.
 */
@friendlyName("NotificationRuleInvoiceCreatedCreateRequest")
@withVisibility(Lifecycle.Create, Lifecycle.Update)
model NotificationRuleInvoiceCreatedCreateRequest {
  ...OmitProperties<NotificationRuleInvoiceCreated, "channels">;

  /**
   * List of notification channels the rule is applied to.
   */
  @visibility(Lifecycle.Create, Lifecycle.Update)
  @summary("Channels")
  @minItems(1)
  channels: Array<ULID>;
}

/**
 * Request with input parameters for creating new notification rule with invoice.updated  type.
 */
@friendlyName("NotificationRuleInvoiceUpdatedCreateRequest")
@withVisibility(Lifecycle.Create, Lifecycle.Update)
model NotificationRuleInvoiceUpdatedCreateRequest {
  ...OmitProperties<NotificationRuleInvoiceUpdated, "channels">;

  /**
   * List of notification channels the rule is applied to.
   */
  @visibility(Lifecycle.Create, Lifecycle.Update)
  @summary("Channels")
  @minItems(1)
  channels: Array<ULID>;
}
