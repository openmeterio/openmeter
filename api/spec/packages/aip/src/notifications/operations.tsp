import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./channel.tsp";
import "./event.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Notifications;

/**
 * Filter options for listing notification channels.
 */
@friendlyName("ListNotificationChannelsParamsFilter")
model ListNotificationChannelsParamsFilter {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  name?: Common.StringFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  type?: Common.StringFieldFilterExact;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  disabled?: Common.BooleanFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  created_at?: Common.DateTimeFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  updated_at?: Common.DateTimeFieldFilter;
}

interface NotificationChannelOperations {
  /**
   * List all notification channels.
   */
  @get
  @operationId("list-notification-channels")
  @summary("List notification channels")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort notification channels returned in the response. Supported sort attributes
     * are:
     *
     * - `id` (default)
     * - `type`
     * - `created_at`
     * - `updated_at`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter notification channels returned in the response.
     *
     * To filter notification channels by type add the following query param:
     * filter[type]=webhook
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListNotificationChannelsParamsFilter,
  ): Shared.PagePaginatedResponse<NotificationChannel> | Common.ErrorResponses;

  /**
   * Create a notification channel.
   */
  @post
  @operationId("create-notification-channel")
  @summary("Create notification channel")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  create(
    @body
    channel: Shared.CreateRequest<NotificationChannel>,
  ): Shared.CreateResponse<NotificationChannel> | Common.ErrorResponses;

  /**
   * Get a notification channel by id.
   */
  @get
  @route("/{notificationChannelId}")
  @operationId("get-notification-channel")
  @summary("Get notification channel")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  get(@path notificationChannelId: Shared.ULID):
    | Shared.GetResponse<NotificationChannel>
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Update a notification channel by id.
   */
  @put
  @route("/{notificationChannelId}")
  @operationId("update-notification-channel")
  @summary("Update notification channel")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  update(
    @path notificationChannelId: Shared.ULID,

    @body
    channel: NotificationChannelUpdateRequest,
  ):
    | Shared.UpdateResponse<NotificationChannel>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Delete a notification channel by id.
   */
  @delete
  @route("/{notificationChannelId}")
  @operationId("delete-notification-channel")
  @summary("Delete notification channel")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  delete(@path notificationChannelId: Shared.ULID):
    | Shared.DeleteResponse
    | Common.NotFound
    | Common.ErrorResponses;
}

/**
 * Filter options for listing notification events.
 */
@friendlyName("ListNotificationEventsParamsFilter")
model ListNotificationEventsParamsFilter {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  id?: Common.ULIDFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  type?: Common.StringFieldFilterExact;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  created_at?: Common.DateTimeFieldFilter;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  rule_id?: Common.ULIDFieldFilter;

  /**
   * Filter by the channels the generating rule targets. This is an existential
   * match: `filter[channel_id][eq]=<id>` returns events whose rule targets that
   * channel, not events that were delivered only to it. Only `eq` and `oeq` are
   * supported; negating an existential match would return events that merely have
   * some other channel as well, so `neq` is rejected.
   */
  channel_id?: Common.ULIDFieldFilter;

  /**
   * Filter by delivery state. This is an existential match:
   * `filter[delivery_status][eq]=failed` returns events where delivery to at least
   * one channel failed, not events where every delivery failed. Only `eq` and `oeq`
   * are supported; negating an existential match would return events that merely
   * have some other state as well, so `neq` is rejected.
   */
  delivery_status?: Common.StringFieldFilterExact;

  /**
   * Filter by the key of the subject the event refers to. Events without a subject
   * are not matched by `neq`.
   */
  subject_key?: Common.StringFieldFilterExact;

  /**
   * Filter by the id of the subject the event refers to. Events without a subject
   * are not matched by `neq`.
   */
  subject_id?: Common.ULIDFieldFilter;

  /**
   * Filter by the key of the feature the event refers to. Events without a feature
   * are not matched by `neq`.
   */
  feature_key?: Common.StringFieldFilterExact;

  /**
   * Filter by the id of the feature the event refers to. Events without a feature
   * are not matched by `neq`.
   */
  feature_id?: Common.ULIDFieldFilter;
}

interface NotificationEventOperations {
  /**
   * List all notification events.
   */
  @get
  @operationId("list-notification-events")
  @summary("List notification events")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort notification events returned in the response. Supported sort attributes
     * are:
     *
     * - `created_at` (default, descending)
     * - `id`
     * - `type`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter notification events returned in the response.
     *
     * To filter notification events by type add the following query param:
     * filter[type]=invoice_created
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListNotificationEventsParamsFilter,
  ): Shared.PagePaginatedResponse<NotificationEvent> | Common.ErrorResponses;

  /**
   * Get a notification event by id.
   */
  @get
  @route("/{notificationEventId}")
  @operationId("get-notification-event")
  @summary("Get notification event")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  get(@path notificationEventId: Shared.ULID):
    | Shared.GetResponse<NotificationEvent>
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Re-send a notification event to the channels of the rule that generated it.
   *
   * Delivery is asynchronous: the request marks the matching delivery statuses for
   * re-delivery and returns immediately. Statuses that are still pending or already
   * being re-sent are left untouched.
   */
  @post
  @route("/{notificationEventId}/resend")
  @operationId("resend-notification-event")
  @summary("Resend notification event")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  resend(
    @path notificationEventId: Shared.ULID,

    @body
    request: NotificationEventResendRequest,
  ): Shared.AcceptedResponse | Common.NotFound | Common.ErrorResponses;
}
