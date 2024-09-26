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
  // @visibility("read")
  @summary("Rule Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * Notification rule type.
   */
  // @visibility("read")
  @summary("Rule Type")
  type: EventType;
}

/**
 * Common fields of a notificaiton channel.
 */
@friendlyName("NotificationRuleCommon")
model RuleCommon<T extends EventType> {
  ...ResourceTimestamps;

  /**
   * Identifies the notification rule.
   */
  // @visibility("read")
  @summary("Rule Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * Notification rule type.
   */
  // @visibility("read", "create", "query")
  @summary("Rule Type")
  type: T;

  /**
   * The user friendly name of the notification rule.
   */
  @summary("Rule Name")
  @example("Balance threshold reached")
  // @visibility("read", "create", "query", "update")
  name: string;

  /**
   * Whether the rule is disabled or not.
   */
  // @visibility("read", "create", "query", "update")
  @summary("Rule Disabled")
  @example(true)
  disabled?: boolean = false;

  /**
   * List of notification channels the rule applies to.
  */
  // @visibility("read")
  @summary("Channels assigned to Rule")
  channels: Array<ChannelMeta>
}

/**
 * Threshold value with multiple supported types.
 */
@friendlyName("NotificationRuleBalanceThresholdValue")
model RuleBalanceThresholdValue {
  /**
   * Value of the threshold.
   */
  // @visibility("read", "create", "update")
  @summary("Threshold Value")
  @example(100)
  value: float64;
  /**
   * Type of the threshold.
   */
  // @visibility("read", "create", "update")
  @summary("Threshold Type")
  @example("NUMBER")
  type: "PERCENT" | "NUMBER";
}

/**
 * Limited representation of a feature resource which includes only its unique identifiers (id, key).
 */
@friendlyName("FeatureMeta")
model FeatureMeta {
  /**
   * Unique identifier of a feature.
   */
  // @visibility("read", "query")
  @summary("Feature Unique Identifier")
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  id: ULID;

  /**
   * The key is an immutable unique identifier of the feature used throughout the API,
   * for example when interacting with a subject's entitlements.
   */
  // @visibility("read", "query")
  @summary("Feature Key")
  @example("gpt4_tokens")
  key: string;
}

/**
 * Notification rule with entitlements.balance.threshold type.
 */
@friendlyName("NotificationRuleBalanceThreshold")
model RuleBalanceThreshold {
  ...RuleCommon<EventType.entitlementsBalanceThreshold>;

  /**
   * List of thresholds the rule suppose to be triggered.
   */
  // @visibility("read", "create", "query", "update")
  @summary("Entitlement Balance Thresholds")
  @minItems(1)
  @maxItems(10)
  thresholds: Array<RuleBalanceThresholdValue>;

  /**
   * Optional field containing list of features the rule applies to.
  */
  // @visibility("read", "query")
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
union Rule {
  entitlementsBalanceThreshold: RuleBalanceThreshold,
}

/**
 * Union type for requests creating new notificatio nchannel with certain type.
 */
@friendlyName("NotificationRuleCreateRequest")
@discriminator("type")
@oneOf
union RuleCreateRequest {
  entitlementsBalanceThreshold: RuleBalanceThresholdCreateRequest,
}

/**
 * Request with input parameters for creating new notification channel with webhook type.
 */
// @withVisibility("create", "update")
@friendlyName("NotificationRuleBalanceThresholdCreateRequest")
model RuleBalanceThresholdCreateRequest{
  ...OmitProperties<RuleBalanceThreshold, "id" | "createdAt" | "updatedAt" | "deletedAt" | "features">;

  /**
   * Optional field for defining the scope of notification by feature. It may contain features by id or key.
   */
  // @visibility("create", "update")
  @summary("Features")
  @minItems(1)
  features?: Array<ULID | string>;
}

/**
 * Order by options for notification channels.
 */
@friendlyName("NotificationRuleOrderBy")
enum RuleOrderBy {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  id: "id",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  type: "type",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  createdAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  updatedAt: "updatedAt",
}

// NOTE(chrisgacsal): this is a workaround of using the model OpenMeter.PaginatedResponse<Rule>
// results a model with friendlyName set to `RulePaginatedResponse` instead of `NotificationRulePaginatedResponse` which
// seems to be a bug in typespec. This way we can override the friendlyName of the paginated response.
/**
 * Paginated response for listing notification rules.
 */
@friendlyName("NotificationRulePaginatedResponse")
model RulesPaginatedResponse is OpenMeter.PaginatedResponse<Rule>;

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
    includeDeleted?: boolean,

    /**
     * Include disabled notification rules in response.
     *
     * Usage: `?includeDisabled=false`
     */
    @query
    @example(false)
    includeDisabled?: boolean,

    /**
     * Filtering by multiple feature ids/keys.
     *
     * Usage: `?feature=feature-1&feature=feature-2`
     */
    @query feature?: Array<ULID | string>,

    /**
     * Filtering by multiple notifiaction channel ids.
     *
     * Usage: `?channel=01ARZ3NDEKTSV4RRFFQ69G5FAV&channel=01J8J2Y5X4NNGQS32CF81W95E3`
     */
    @query channel?: Array<ULID | string>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<RuleOrderBy>,
  ): RulesPaginatedResponse | OpenMeter.CommonErrors;

  /**
   * Create a new notification rule.
   */
  @post
  @operationId("createNotificationRule")
  @summary("Create a notification rule")
  create(@body request: RuleCreateRequest): {
    @statusCode _: 201;
    @body body: Rule;
  } | OpenMeter.CommonErrors;

  /**
   * Update notification rule.
   */
  @put
  @operationId("updateNotificationRule")
  @summary("Update a notification rule")
  update(@body request: RuleCreateRequest): {
    @statusCode _: 200;
    @body body: Rule;
  } | OpenMeter.CommonErrors;

  /**
   * Get a notification rule by id.
   */
  @get
  @operationId("getNotificationRule")
  @summary("Get notification rule")
  get(@path ruleId: ULID): Rule | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

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
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
