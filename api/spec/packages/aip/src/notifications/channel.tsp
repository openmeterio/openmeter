import "../shared/index.tsp";

namespace Notifications;

/**
 * The type of a notification channel.
 *
 * Currently the only supported channel type is `webhook`; the domain model
 * anticipates additional channel types (e.g. email, Slack) in the future, so this
 * is modeled as an enum rather than a boolean/constant even though it has a single
 * member today.
 */
@friendlyName("NotificationChannelType")
enum NotificationChannelType {
  /**
   * A webhook channel delivers notification events as signed HTTP POST
   * requests to a configured URL.
   */
  Webhook: "webhook",
}

/**
 * A notification channel delivers notification events, such as entitlement balance
 * threshold crossings, to an external system. Today the only supported channel
 * type is a webhook delivered via Svix.
 */
@friendlyName("NotificationChannel")
model NotificationChannel {
  ...OmitProperties<Shared.Resource, "description">;

  /**
   * The type of the channel. Immutable after creation.
   */
  @visibility(Lifecycle.Read, Lifecycle.Update, Lifecycle.Create)
  type: NotificationChannelType;

  /**
   * Whether the channel is disabled. Disabled channels do not receive notification
   * events.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  disabled?: boolean = false;

  /**
   * The URL that webhook notification events are delivered to.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  url: string;

  /**
   * Custom HTTP headers to include on every webhook delivery request, keyed by
   * header name.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  custom_headers?: Record<string>;

  /**
   * Secret used to sign outgoing webhook payloads so recipients can verify their
   * authenticity. If omitted on create, a secret is generated automatically by the
   * delivery provider. This is a sensitive credential returned in responses (unlike
   * most secrets) specifically so clients can retrieve a server-generated value and
   * verify webhook signatures; handle it with the same care as any other credential.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  signing_secret?: string;
}

/**
 * Request body for updating a notification channel. Updates replace the channel's
 * mutable state rather than merging it: `type`, `name`, and `url` must always be
 * provided, and omitting `disabled`, `labels`, or `custom_headers` resets them to
 * their defaults (enabled, no labels, no custom headers). `signing_secret` is the
 * one exception: omitting it keeps the channel's current signing secret instead of
 * clearing the credential.
 */
@friendlyName("UpdateNotificationChannelRequest")
model NotificationChannelUpdateRequest {
  ...PickProperties<
    UpdateableProperties<NotificationChannel>,
    "type" | "name" | "url"
  >;
  ...OptionalProperties<PickProperties<
    UpdateableProperties<NotificationChannel>,
    "disabled" | "custom_headers" | "signing_secret" | "labels"
  >>;
}
