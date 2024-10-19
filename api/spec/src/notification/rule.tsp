import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Notification;

/**
 * Metadata only fields of a notification channel.
 */
@friendlyName("NotificationRuleMeta")
model NotificationRuleMeta {
  /**
   * Identifies the notification rule.
   */
  @visibility("read")
  @summary("Rule Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * Notification rule type.
   */
  @visibility("read")
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
  @visibility("read", "create", "query")
  @summary("Rule Type")
  type: T;

  /**
   * The user friendly name of the notification rule.
   */
  @summary("Rule Name")
  @example("Balance threshold reached")
  @visibility("read", "create", "query", "update")
  name: string;

  /**
   * Whether the rule is disabled or not.
   */
  @visibility("read", "create", "query", "update")
  @summary("Rule Disabled")
  @example(true)
  disabled?: boolean = false;

  /**
   * List of notification channels the rule applies to.
   */
  @visibility("read", "create", "query", "update")
  @summary("Channels assigned to Rule")
  channels: Array<NotificationChannelMeta>;
}

/**
 * Threshold value with multiple supported types.
 */
@friendlyName("NotificationRuleBalanceThresholdValue")
model NotificationRuleBalanceThresholdValue {
  /**
   * Value of the threshold.
   */
  @visibility("read", "create", "update")
  @summary("Threshold Value")
  @example(100)
  value: float64;

  /**
   * Type of the threshold.
   */
  @visibility("read", "create", "update")
  @example(NotificationRuleBalanceThresholdValueType.NUMBER)
  type: NotificationRuleBalanceThresholdValueType;
}

/**
 * Type of the rule in the balance threshold specification.
 */
@summary("Notification balance threshold type")
@friendlyName("NotificationRuleBalanceThresholdValueType")
@extension("x-enum-varnames", ["Percent", "Number"])
enum NotificationRuleBalanceThresholdValueType {
  PERCENT,
  NUMBER,
}

/**
 * Limited representation of a feature resource which includes only its unique identifiers (id, key).
 */
@friendlyName("FeatureMeta")
model FeatureMeta {
  /**
   * Unique identifier of a feature.
   */
  @visibility("read", "query")
  @summary("Feature Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * The key is an immutable unique identifier of the feature used throughout the API,
   * for example when interacting with a subject's entitlements.
   */
  @visibility("read", "query")
  @summary("Feature Key")
  @example("gpt4_tokens")
  key: string;
}

/**
 * Notification rule with entitlements.balance.threshold type.
 */
@friendlyName("NotificationRuleBalanceThreshold")
model NotificationRuleBalanceThreshold {
  ...NotificationRuleCommon<NotificationEventType.entitlementsBalanceThreshold>;

  /**
   * List of thresholds the rule suppose to be triggered.
   */
  @visibility("read", "create", "query", "update")
  @summary("Entitlement Balance Thresholds")
  @minItems(1)
  @maxItems(10)
  thresholds: Array<NotificationRuleBalanceThresholdValue>;

  /**
   * Optional field containing list of features the rule applies to.
   */
  @visibility("read", "query")
  @summary("Features")
  @minItems(1)
  features?: Array<FeatureMeta>;
}

/**
 * Notification Rule.
 */
@friendlyName("NotificationRule")
@discriminator("type")
@oneOf
union NotificationRule {
  entitlementsBalanceThreshold: NotificationRuleBalanceThreshold,
}

/**
 * Union type for requests creating new notification rule with certain type.
 */
@friendlyName("NotificationRuleCreateRequest")
@discriminator("type")
@oneOf
union NotificationRuleCreateRequest {
  entitlementsBalanceThreshold: NotificationRuleBalanceThresholdCreateRequest,
}

/**
 * Request with input parameters for creating new notification rule with entitlements.balance.threashold type.
 */
@withVisibility("create", "update")
@friendlyName("NotificationRuleBalanceThresholdCreateRequest")
model NotificationRuleBalanceThresholdCreateRequest {
  ...OmitProperties<NotificationRuleBalanceThreshold, "channels" | "features">;

  /**
   * List of notification channels the rule is applied to.
   */
  @visibility("create", "update")
  @summary("Channels")
  @minItems(1)
  channels: Array<ULID>;

  /**
   * Optional field for defining the scope of notification by feature. It may contain features by id or key.
   */
  @visibility("create", "update")
  @summary("Features")
  @minItems(1)
  features?: Array<ULIDOrKey>;
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
@tag("Notification (Experimental)")
interface Rules {
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

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<NotificationRuleOrderBy>,
  ): OpenMeter.PaginatedResponse<NotificationRule> | OpenMeter.CommonErrors;

  /**
   * Create a new notification rule.
   */
  @post
  @operationId("createNotificationRule")
  @summary("Create a notification rule")
  create(@body request: NotificationRuleCreateRequest): {
    @statusCode _: 201;
    @body body: NotificationRule;
  } | OpenMeter.CommonErrors;

  /**
   * Update notification rule.
   */
  @put
  @operationId("updateNotificationRule")
  @summary("Update a notification rule")
  update(@path ruleId: ULID, @body request: NotificationRuleCreateRequest): {
    @statusCode _: 200;
    @body body: NotificationRule;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a notification rule by id.
   */
  @get
  @operationId("getNotificationRule")
  @summary("Get notification rule")
  get(
    @path ruleId: ULID,
  ): NotificationRule | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

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
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Test a notification rule by sending a test event with random data.
   */
  @post
  @route("/{ruleId}/test")
  @operationId("testNotificationRule")
  test(@path ruleId: ULID): {
    @statusCode _: 201;
    @body body: NotificationEvent;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
