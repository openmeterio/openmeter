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
import "./rule.tsp";

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
 * Filter options for listing notification rules.
 */
@friendlyName("ListNotificationRulesParamsFilter")
model ListNotificationRulesParamsFilter {
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

  /**
   * Filter by an assigned channel. Matches rules that deliver to at least one of the
   * given channels. Only `eq` and `oeq` are supported.
   */
  channel_id?: Common.ULIDFieldFilter;
}

interface NotificationRuleOperations {
  /**
   * List all notification rules.
   */
  @get
  @operationId("list-notification-rules")
  @summary("List notification rules")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort notification rules returned in the response. Supported sort attributes are:
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
     * Filter notification rules returned in the response.
     *
     * Examples:
     *
     * - `filter[type]=entitlements.balance.threshold`
     * - `filter[disabled]=false`
     * - `filter[channel_id]=01ARZ3NDEKTSV4RRFFQ69G5FAV`
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListNotificationRulesParamsFilter,
  ): Shared.PagePaginatedResponse<NotificationRule> | Common.ErrorResponses;

  /**
   * Create a notification rule.
   */
  @post
  @operationId("create-notification-rule")
  @summary("Create notification rule")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  create(
    @body
    rule: NotificationRuleRequest,
  ): Shared.CreateResponse<NotificationRule> | Common.ErrorResponses;

  /**
   * Get a notification rule by id.
   */
  @get
  @route("/{notificationRuleId}")
  @operationId("get-notification-rule")
  @summary("Get notification rule")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  get(@path notificationRuleId: Shared.ULID):
    | Shared.GetResponse<NotificationRule>
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Update a notification rule by id.
   */
  @put
  @route("/{notificationRuleId}")
  @operationId("update-notification-rule")
  @summary("Update notification rule")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  update(
    @path notificationRuleId: Shared.ULID,

    @body
    rule: NotificationRuleRequest,
  ):
    | Shared.UpdateResponse<NotificationRule>
    | Common.NotFound
    | Common.Conflict
    | Common.ErrorResponses;

  /**
   * Delete a notification rule by id.
   */
  @delete
  @route("/{notificationRuleId}")
  @operationId("delete-notification-rule")
  @summary("Delete notification rule")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  delete(@path notificationRuleId: Shared.ULID):
    | Shared.DeleteResponse
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Test a notification rule by generating an event with sample data and delivering
   * it to the rule's channels. The test event is persisted and listed like any other
   * event.
   */
  @post
  @route("/{notificationRuleId}/test")
  @operationId("test-notification-rule")
  @summary("Test notification rule")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.RPCEndpointExtension, true)
  test(@path notificationRuleId: Shared.ULID):
    | Shared.CreateResponse<NotificationEvent>
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
   * Filter by a channel of the generating rule. Matches events whose rule targets at
   * least one of the given channels. Only `eq` and `oeq` are supported.
   */
  channel_id?: Common.ULIDFieldFilter;

  /**
   * Filter by delivery state. Matches events where delivery to at least one channel
   * is in one of the given states. Only `eq` and `oeq` are supported.
   */
  delivery_status?: Common.StringFieldFilterExact;

  /**
   * Filter by the key of the subject the event refers to. Events without a subject
   * never match, not even with `neq`.
   */
  subject_key?: Common.StringFieldFilterExact;

  /**
   * Filter by the id of the subject the event refers to. Events without a subject
   * never match, not even with `neq`.
   */
  subject_id?: Common.ULIDFieldFilter;

  /**
   * Filter by the key of the feature the event refers to. Events without a feature
   * never match, not even with `neq`.
   */
  feature_key?: Common.StringFieldFilterExact;

  /**
   * Filter by the id of the feature the event refers to. Events without a feature
   * never match, not even with `neq`.
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
     * - `created_at` (default)
     * - `id`
     * - `type`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order. Without a `sort` parameter, events
     * are returned newest first.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter notification events returned in the response.
     *
     * Examples:
     *
     * - `filter[type]=invoice.created`
     * - `filter[delivery_status]=failed`
     * - `filter[subject_key]=customer-1`
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
   * Resend a notification event to the channels of the rule that generated it.
   *
   * Delivery is asynchronous: the request marks the selected channels for redelivery
   * and returns immediately. Channels whose delivery is still pending or already
   * being resent are left untouched.
   */
  @post
  @route("/{notificationEventId}/resend")
  @operationId("resend-notification-event")
  @summary("Resend notification event")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.RPCEndpointExtension, true)
  resend(
    @path notificationEventId: Shared.ULID,

    @body
    request: NotificationEventResendRequest,
  ): Shared.AcceptedResponse | Common.NotFound | Common.ErrorResponses;
}
