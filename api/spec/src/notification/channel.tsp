import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Notification;

/**
 * Type of the notification channel.
 */
@friendlyName("NotificationChannelType")
enum NotificationChannelType {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  webhook: "WEBHOOK",
}

/**
 * Metadata only fields of a notification channel.
 */
@friendlyName("NotificationChannelMeta")
model NotificationChannelMeta {
  /**
   * Identifies the notification channel.
   */
  @visibility("read")
  @summary("Channel Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * Notification channel type.
   */
  @visibility("read")
  @summary("Channel Type")
  type: NotificationChannelType;
}

/**
 * Common fields of a notificaiton channel.
 */
@friendlyName("NotificationChannelCommon")
model NotificationChannelCommon<T extends NotificationChannelType> {
  ...ResourceTimestamps;
  ...OmitProperties<NotificationChannelMeta, "type">;

  /**
   * Notification channel type.
   */
  @visibility("read", "create")
  @summary("Channel Type")
  type: T;

  /**
   * User friendly name of the channel.
   */
  @visibility("read", "create", "update")
  @summary("Channel Name")
  @example("customer-webhook")
  name: string;

  /**
   * Whether the channel is disabled or not.
   */
  @visibility("read", "create", "update")
  @summary("Channel Disabled")
  @example(true)
  disabled?: boolean = false;
}

/**
 * Notification channel with webhook type.
 */
@friendlyName("NotificationChannelWebhook")
model NotificationChannelWebhook {
  ...NotificationChannelCommon<NotificationChannelType.webhook>;

  /**
   * Webhook URL where the notification is sent.
   */
  @visibility("read", "create", "update")
  @summary("Webhook URL")
  @example("https://example.com/webhook")
  url: string;

  /**
   * Custom HTTP headers sent as part of the webhook request.
   */
  @visibility("read", "create", "update")
  @summary("Custom HTTP Headers")
  customHeaders?: Record<string>;

  /**
   * Signing secret used for webhook request validation on the receiving end.
   *
   * Format: `base64` encoded random bytes optionally prefixed with `whsec_`. Recommended size: 24
   */
  @visibility("read", "create", "update")
  @summary("Signing Secret")
  @pattern("^(whsec_)?[a-zA-Z0-9+/=]{32,100}$")
  @example("whsec_S6g2HLnTwd9AhHwUIMFggVS9OfoPafN8")
  signingSecret?: string;
}

/**
 * Notification channel.
 */
@friendlyName("NotificationChannel")
@discriminator("type")
@oneOf
union NotificationChannel {
  webhook: NotificationChannelWebhook,
}

/**
 * Union type for requests creating new notification channel with certain type.
 */
@friendlyName("NotificationChannelCreateRequest")
@discriminator("type")
@oneOf
union NotificationChannelCreateRequest {
  webhook: NotificationChannelWebhookCreateRequest,
}

/**
 * Request with input parameters for creating new notification channel with webhook type.
 */
@withVisibility("create", "update")
@friendlyName("NotificationChannelWebhookCreateRequest")
model NotificationChannelWebhookCreateRequest is NotificationChannelWebhook;

/**
 * Order by options for notification channels.
 */
@friendlyName("NotificationChannelOrderBy")
enum NotificationChannelOrderBy {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  id: "id",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  type: "type",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  createdAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  updatedAt: "updatedAt",
}

@route("/api/v1/notification/channels")
@tag("Notification (Experimental)")
interface Channels {
  /**
   * List all notification channels.
   */
  @get
  @operationId("listNotificationChannels")
  @summary("List notification channels")
  list(
    /**
     * Include deleted notification channels in response.
     *
     * Usage: `?includeDeleted=true`
     */
    @query
    @example(true)
    includeDeleted?: boolean = false,

    /**
     * Include disabled notification channels in response.
     *
     * Usage: `?includeDisabled=false`
     */
    @query
    @example(false)
    includeDisabled?: boolean = false,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<NotificationChannelOrderBy>,
  ): OpenMeter.PaginatedResponse<NotificationChannel> | OpenMeter.CommonErrors;

  /**
   * Create a new notification channel.
   */
  @post
  @operationId("createNotificationChannel")
  @summary("Create a notification channel")
  create(@body request: NotificationChannelCreateRequest): {
    @statusCode _: 201;
    @body body: NotificationChannel;
  } | OpenMeter.CommonErrors;

  /**
   * Update notification channel.
   */
  @put
  @operationId("updateNotificationChannel")
  @summary("Update a notification channel")
  update(@path channelId: ULID, @body request: NotificationChannelCreateRequest): {
    @statusCode _: 200;
    @body body: NotificationChannel;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a notification channel by id.
   */
  @get
  @operationId("getNotificationChannel")
  @summary("Get notification channel")
  get(@path channelId: ULID): NotificationChannel | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Soft delete notification channel by id.
   *
   * Once a notification channel is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deleteNotificationChannel")
  @summary("Delete a notification channel")
  delete(@path channelId: ULID): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
