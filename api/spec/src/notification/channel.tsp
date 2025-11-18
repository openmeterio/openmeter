import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Type of the notification channel.
 */
@friendlyName("NotificationChannelType")
@extension("x-enum-varnames", #["Webhook"])
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
  @visibility(Lifecycle.Read)
  @summary("Channel Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * Notification channel type.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
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
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Channel Type")
  type: T;

  /**
   * User friendly name of the channel.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Channel Name")
  @example("customer-webhook")
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Whether the channel is disabled or not.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Channel Disabled")
  @example(true)
  disabled?: boolean = false;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: Annotations;

  /**
   * Additional metadata for the resource.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Metadata")
  metadata?: Metadata | null;
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
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Webhook URL")
  @example("https://example.com/webhook")
  url: string;

  /**
   * Custom HTTP headers sent as part of the webhook request.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Custom HTTP Headers")
  customHeaders?: Record<string>;

  /**
   * Signing secret used for webhook request validation on the receiving end.
   *
   * Format: `base64` encoded random bytes optionally prefixed with `whsec_`. Recommended size: 24
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Signing Secret")
  @pattern("^(whsec_)?[a-zA-Z0-9+/=]{32,100}$")
  @example("whsec_S6g2HLnTwd9AhHwUIMFggVS9OfoPafN8")
  signingSecret?: string;
}

/**
 * Notification channel.
 */
@friendlyName("NotificationChannel")
union NotificationChannel {
  NotificationChannelWebhook,
}

/**
 * Union type for requests creating new notification channel with certain type.
 */
@friendlyName("NotificationChannelCreateRequest")
union NotificationChannelCreateRequest {
  NotificationChannelWebhookCreateRequest,
}

/**
 * Request with input parameters for creating new notification channel with webhook type.
 */
@friendlyName("NotificationChannelWebhookCreateRequest")
model NotificationChannelWebhookCreateRequest
  is TypeSpec.Rest.Resource.ResourceCreateModel<NotificationChannelWebhook>;

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
@tag("Notifications")
@friendlyName("NotificationChannels")
interface NotificationChannelsEndpoints {
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

    ...QueryPagination,
    ...QueryOrdering<NotificationChannelOrderBy>,
  ): PaginatedResponse<NotificationChannel> | CommonErrors;

  /**
   * Create a new notification channel.
   */
  @post
  @operationId("createNotificationChannel")
  @summary("Create a notification channel")
  create(@body request: NotificationChannelCreateRequest): {
    @statusCode _: 201;
    @body body: NotificationChannel;
  } | CommonErrors;

  /**
   * Update notification channel.
   */
  @put
  @operationId("updateNotificationChannel")
  @summary("Update a notification channel")
  update(
    @path channelId: ULID,
    @body request: NotificationChannelCreateRequest,
  ): {
    @statusCode _: 200;
    @body body: NotificationChannel;
  } | NotFoundError | CommonErrors;

  /**
   * Get a notification channel by id.
   */
  @get
  @operationId("getNotificationChannel")
  @summary("Get notification channel")
  get(@path channelId: ULID): {
    @statusCode _: 200;
    @body body: NotificationChannel;
  } | NotFoundError | CommonErrors;

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
  } | NotFoundError | CommonErrors;
}
