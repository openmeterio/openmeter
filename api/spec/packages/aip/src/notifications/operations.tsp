import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./channel.tsp";

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
