import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Metadata only fields of a notification channel.
 */
@friendlyName("NotificationRuleMeta")
model NotificationRuleMeta {
  /**
   * Identifies the notification rule.
   */
  @visibility(Lifecycle.Read)
  @summary("Rule Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * Notification rule type.
   */
  @visibility(Lifecycle.Read)
  @summary("Rule Type")
  type: NotificationEventType;
}

/**
 * Common fields of a notificaiton channel.
 */
@friendlyName("NotificationRuleCommon")
model NotificationRuleCommon<T extends NotificationEventType> {
  ...ResourceTimestamps;
  ...OmitProperties<NotificationRuleMeta, "type">;

  /**
   * Notification rule type.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rule Type")
  type: T;

  /**
   * The user friendly name of the notification rule.
   */
  @summary("Rule Name")
  @example("Balance threshold reached")
  @minLength(1)
  @maxLength(256)
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  name: string;

  /**
   * Whether the rule is disabled or not.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Rule Disabled")
  @example(true)
  disabled?: boolean = false;

  /**
   * List of notification channels the rule applies to.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Channels assigned to Rule")
  channels: Array<NotificationChannelMeta>;

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
 * Notification Rule.
 */
@friendlyName("NotificationRule")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union NotificationRule {
  `entitlements.balance.threshold`: NotificationRuleBalanceThreshold,
  `entitlements.reset`: NotificationRuleEntitlementReset,
  `invoice.created`: NotificationRuleInvoiceCreated,
  `invoice.updated`: NotificationRuleInvoiceUpdated,
}

/**
 * Union type for requests creating new notification rule with certain type.
 */
@friendlyName("NotificationRuleCreateRequest")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union NotificationRuleCreateRequest {
  `entitlements.balance.threshold`: NotificationRuleBalanceThresholdCreateRequest,
  `entitlements.reset`: NotificationRuleEntitlementResetCreateRequest,
  `invoice.created`: NotificationRuleInvoiceCreatedCreateRequest,
  `invoice.updated`: NotificationRuleInvoiceUpdatedCreateRequest,
}

/**
 * Order by options for notification channels.
 */
@friendlyName("NotificationRuleOrderBy")
enum NotificationRuleOrderBy {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  id: "id",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  type: "type",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  createdAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  updatedAt: "updatedAt",
}

@route("/api/v1/notification/rules")
@tag("Notifications")
@friendlyName("NotificationRules")
interface NotificationRulesEndpoints {
  /**
   * List all notification rules.
   */
  @get
  @operationId("listNotificationRules")
  @summary("List notification rules")
  list(
    /**
     * Include deleted notification rules in response.
     *
     * Usage: `?includeDeleted=true`
     */
    @query
    @example(true)
    includeDeleted?: boolean = false,

    /**
     * Include disabled notification rules in response.
     *
     * Usage: `?includeDisabled=false`
     */
    @query
    @example(false)
    includeDisabled?: boolean = false,

    /**
     * Filtering by multiple feature ids/keys.
     *
     * Usage: `?feature=feature-1&feature=feature-2`
     */
    @query(#{ explode: true })
    feature?: Array<ULIDOrKey>,

    /**
     * Filtering by multiple notifiaction channel ids.
     *
     * Usage: `?channel=01ARZ3NDEKTSV4RRFFQ69G5FAV&channel=01J8J2Y5X4NNGQS32CF81W95E3`
     */
    @query(#{ explode: true })
    channel?: Array<string>,

    ...QueryPagination,
    ...QueryOrdering<NotificationRuleOrderBy>,
  ): PaginatedResponse<NotificationRule> | CommonErrors;

  /**
   * Create a new notification rule.
   */
  @post
  @operationId("createNotificationRule")
  @summary("Create a notification rule")
  create(@body request: NotificationRuleCreateRequest): {
    @statusCode _: 201;
    @body body: NotificationRule;
  } | CommonErrors;

  /**
   * Update notification rule.
   */
  @put
  @operationId("updateNotificationRule")
  @summary("Update a notification rule")
  update(@path ruleId: ULID, @body request: NotificationRuleCreateRequest): {
    @statusCode _: 200;
    @body body: NotificationRule;
  } | NotFoundError | CommonErrors;

  /**
   * Get a notification rule by id.
   */
  @get
  @operationId("getNotificationRule")
  @summary("Get notification rule")
  get(@path ruleId: ULID): NotificationRule | NotFoundError | CommonErrors;

  /**
   * Soft delete notification rule by id.
   *
   * Once a notification rule is deleted it cannot be undeleted.
   */
  @delete
  @operationId("deleteNotificationRule")
  @summary("Delete a notification rule")
  delete(@path ruleId: ULID): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;

  /**
   * Test a notification rule by sending a test event with random data.
   */
  @post
  @route("/{ruleId}/test")
  @operationId("testNotificationRule")
  @summary("Test notification rule")
  test(@path ruleId: ULID): {
    @statusCode _: 201;
    @body body: NotificationEvent;
  } | NotFoundError | CommonErrors;
}
