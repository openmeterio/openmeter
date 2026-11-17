import "../shared/index.tsp";

namespace Notifications;

/**
 * A versioned kind of notification event. Its contract covers the payload and,
 * when rules can produce it, the rule configuration.
 */
@friendlyName("BillingNotificationEventTypeSummary")
model NotificationEventTypeSummary {
  /**
   * The canonical event type, `v<version>.<name>`, for example
   * `v1.billing.invoice.created`.
   */
  @visibility(Lifecycle.Read)
  type: string;

  /**
   * What the event type means, including the legacy name it replaces, if any.
   */
  @visibility(Lifecycle.Read)
  description: string;

  /**
   * The event type that replaces this one. Deprecated event types stay usable.
   * Absent when the event type is not deprecated.
   */
  @visibility(Lifecycle.Read)
  deprecated_by?: string;
}

/**
 * A notification event type with the JSON Schemas of its contract.
 */
@friendlyName("BillingNotificationEventTypeDefinition")
model NotificationEventTypeDefinition {
  ...NotificationEventTypeSummary;

  /**
   * The JSON Schema (draft 2020-12) of the event payload.
   */
  @visibility(Lifecycle.Read)
  payload_json_schema: Record<unknown>;

  /**
   * The JSON Schema (draft 2020-12) of the rule configuration. Absent when no rule
   * can produce the event type.
   */
  @visibility(Lifecycle.Read)
  rule_config_json_schema?: Record<unknown>;
}
