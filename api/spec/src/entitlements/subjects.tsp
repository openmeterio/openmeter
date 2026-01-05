import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/subjects/{subjectIdOrKey}/entitlements")
@tag("Entitlements")
@friendlyName("Subjects")
interface SubjectEntitlementsEndpoints {
  /**
   * OpenMeter has three types of entitlements: metered, boolean, and static. The type property determines the type of entitlement. The underlying feature has to be compatible with the entitlement type specified in the request (e.g., a metered entitlement needs a feature associated with a meter).
   *
   * - Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
   * - Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config).
   * - Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.
   *
   * A given subject can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.
   *
   * Once an entitlement is created you cannot modify it, only delete it.
   *
   * ⚠️ __Deprecated__: Use [`POST /api/v2/customers/{customerIdOrKey}/entitlements`](#tag/entitlements/post/api/v2/customers/{customerIdOrKey}/entitlements) instead.
   */
  #deprecated "Use `POST /api/v1/customers/{customerIdOrKey}/entitlements` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @post
  @summary("Create a subject entitlement")
  @operationId("createEntitlement")
  post(
    @path subjectIdOrKey: string,
    @body entitlement: EntitlementCreateInputs,
  ): {
    @statusCode _: 201;
    @body body: Entitlement;
  } | CommonErrors | ConflictError;

  /**
   * List all entitlements for a subject. For checking entitlement access, use the /value endpoint instead.
   *
   * ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements) instead.
   */
  #deprecated "Use `GET /api/v2/customers/{customerIdOrKey}/entitlements` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @get
  @operationId("listSubjectEntitlements")
  @summary("List subject entitlements")
  list(
    @path subjectIdOrKey: string,
    @query includeDeleted?: boolean = false,
  ): Entitlement[] | CommonErrors;

  /**
   * Get entitlement by id. For checking entitlement access, use the /value endpoint instead.
   *
   * ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}) instead.
   */
  #deprecated "Use `GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @get
  @operationId("getEntitlement")
  @summary("Get subject entitlement")
  get(
    @path subjectIdOrKey: string,
    @path entitlementId: string,
  ): Entitlement | CommonErrors | NotFoundError;

  /**
   * Deleting an entitlement revokes access to the associated feature. As a single subject can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well.
   * As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.
   *
   * ⚠️ __Deprecated__: Use [`DELETE /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}`](#tag/entitlements/delete/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}) instead.
   */
  #deprecated "Use `DELETE /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @delete
  @operationId("deleteEntitlement")
  @summary("Delete subject entitlement")
  delete(@path subjectIdOrKey: string, @path entitlementId: string): {
    @statusCode _: 204;
  } | CommonErrors | NotFoundError;

  /**
   * Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided subject-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.
   *
   * This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.
   *
   * ⚠️ __Deprecated__: Use [`PUT /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override`](#tag/entitlements/put/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override) instead.
   */
  #deprecated "Use `PUT /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/override` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @put
  @operationId("overrideEntitlement")
  @route("/{entitlementIdOrFeatureKey}/override")
  @summary("Override subject entitlement")
  override(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @body entitlement: EntitlementCreateInputs,
  ):
    | {
        @statusCode _: 201;
        @body body: Entitlement;
      }
    | CommonErrors
    | ConflictError
    | NotFoundError;

  /**
   * List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.
   *
   * ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants) instead.
   */
  #deprecated "Use `GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @get
  @operationId("listEntitlementGrants")
  @route("/{entitlementIdOrFeatureKey}/grants")
  @summary("List subject entitlement grants")
  getGrants(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @query includeDeleted?: boolean = false,
    @query orderBy?: GrantOrderBy = GrantOrderBy.UpdatedAt,
  ): Grant[] | CommonErrors;

  /**
   * Grants define a behavior of granting usage for a metered entitlement. They can have complicated recurrence and rollover rules, thanks to which you can define a wide range of access patterns with a single grant, in most cases you don't have to periodically create new grants. You can only issue grants for active metered entitlements.
   *
   * A grant defines a given amount of usage that can be consumed for the entitlement. The grant is in effect between its effective date and its expiration date. Specifying both is mandatory for new grants.
   *
   * Grants have a priority setting that determines their order of use. Lower numbers have higher priority, with 0 being the highest priority.
   *
   * Grants can have a recurrence setting intended to automate the manual reissuing of grants. For example, a daily recurrence is equal to reissuing that same grant every day (ignoring rollover settings).
   *
   * Rollover settings define what happens to the remaining balance of a grant at a reset. Balance_After_Reset = MIN(MaxRolloverAmount, MAX(Balance_Before_Reset, MinRolloverAmount))
   *
   * Grants cannot be changed once created, only deleted. This is to ensure that balance is deterministic regardless of when it is queried.
   *
   * ⚠️ __Deprecated__: Use [`POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants`](#tag/entitlements/post/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants) instead.
   */
  #deprecated "Use `POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @post
  @operationId("createGrant")
  @route("/{entitlementIdOrFeatureKey}/grants")
  @summary("Create subject entitlement grant")
  createGrant(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @body grant: GrantCreateInput,
  ): {
    @statusCode _: 201;
    @body body: Grant;
  } | CommonErrors | ConflictError;

  /**
   * This endpoint should be used for access checks and enforcement. All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
   *
   * For convenience reasons, /value works with both entitlementId and featureKey.
   *
   * ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value) instead.
   */
  #deprecated "Use `GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/value` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @get
  @operationId("getEntitlementValue")
  @route("/{entitlementIdOrFeatureKey}/value")
  @summary("Get subject entitlement value")
  getEntitlementValue(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @query(#{ explode: true }) time?: DateTime,
  ): EntitlementValue | CommonErrors | NotFoundError;

  /**
   * Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.
   *
   * BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.
   *
   * WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.
   *
   * ⚠️ __Deprecated__: Use [`GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history`](#tag/entitlements/get/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history) instead.
   */
  #deprecated "Use `GET /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/history` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @get
  @operationId("getEntitlementHistory")
  @route("/{entitlementId}/history")
  @summary("Get subject entitlement history")
  getEntitlementHistory(
    @path subjectIdOrKey: string,
    @path entitlementId: string,

    /**
     * Start of time range to query entitlement: date-time in RFC 3339 format. Defaults to the last reset. Gets truncated to the granularity of the underlying meter.
     */
    @query(#{ explode: true }) from?: DateTime,

    /**
     * End of time range to query entitlement: date-time in RFC 3339 format. Defaults to now.
     * If not now then gets truncated to the granularity of the underlying meter.
     */
    @query(#{ explode: true }) to?: DateTime,

    /**
     * Windowsize
     */
    @query windowSize: WindowSize,

    /**
     * The timezone used when calculating the windows.
     */
    @query windowTimeZone?: string = "UTC",
  ): WindowedBalanceHistory | CommonErrors | NotFoundError;

  /**
   * Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the subjects billing period to enforce usage based on their subscription.
   *
   * Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.
   *
   * ⚠️ __Deprecated__: Use [`POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset`](#tag/entitlements/post/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset) instead.
   */
  #deprecated "Use `POST /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/reset` instead"
  #suppress "deprecated" "Subjects APIs will be removed on December 1st, 2025"
  @post
  @operationId("resetEntitlementUsage")
  @route("/{entitlementId}/reset")
  @summary("Reset subject entitlement")
  reset(
    @path subjectIdOrKey: string,
    @path entitlementId: string,
    @body reset: ResetEntitlementUsageInput,
  ): {
    @statusCode _: 204;
  } | CommonErrors | NotFoundError;
}

/**
 * Entitlements are the core of OpenMeter access management. They define access to features for subjects. Entitlements can be metered, boolean, or static.
 */
@friendlyName("EntitlementValue")
model EntitlementValue {
  /**
   * Whether the subject has access to the feature. Shared accross all entitlement types.
   */
  @visibility(Lifecycle.Read)
  @example(true)
  hasAccess: boolean;

  /**
   * Only available for metered entitlements. Metered entitlements are built around a balance calculation where feature usage is deducted from the issued grants. Balance represents the remaining balance of the entitlement, it's value never turns negative.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  balance?: float64;

  /**
   * Only available for metered entitlements. Returns the total feature usage in the current period.
   */
  @example(50)
  @visibility(Lifecycle.Read)
  usage?: float64;

  /**
   * Only available for metered entitlements. Overage represents the usage that wasn't covered by grants, e.g. if the subject had a total feature usage of 100 in the period but they were only granted 80, there would be 20 overage.
   */
  @example(0)
  @visibility(Lifecycle.Read)
  overage?: float64;

  /**
   * Only available for metered entitlements. The summed amount of all grant active at query time PLUS the used amount of since inactive grants.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  totalAvailableGrantAmount?: float64;

  /**
   * Only available for static entitlements. The JSON parsable config of the entitlement.
   */
  @example("{ key: \"value\" }")
  @visibility(Lifecycle.Read)
  config?: string;
}

/**
 * The windowed balance history.
 */
@friendlyName("WindowedBalanceHistory")
model WindowedBalanceHistory {
  /**
   *  The windowed balance history.
   * - It only returns rows for windows where there was usage.
   * - The windows are inclusive at their start and exclusive at their end.
   * - The last window may be smaller than the window size and is inclusive at both ends.
   */
  windowedHistory: BalanceHistoryWindow[];

  /**
   * Grant burndown history.
   */
  burndownHistory: GrantBurnDownHistorySegment[];
}

/**
 * The balance history window.
 */
@friendlyName("BalanceHistoryWindow")
model BalanceHistoryWindow {
  period: Period;

  /**
   * The total usage of the feature in the period.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  usage: float64;

  /**
   * The entitlement balance at the start of the period.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  balanceAtStart: float64;
}

/**
 *  A segment of the grant burn down history.
 *
 * A given segment represents the usage of a grant between events that changed either the grant burn down priority order or the usag period.
 */
@friendlyName("GrantBurnDownHistorySegment")
model GrantBurnDownHistorySegment {
  /**
   * The period of the segment.
   */
  period: Period;

  /**
   * The total usage of the grant in the period.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  usage: float64;

  /**
   * Overuse that wasn't covered by grants.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  overage: float64;

  /**
   * entitlement balance at the start of the period.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  balanceAtStart: float64;

  /**
   * The balance breakdown of each active grant at the start of the period: GrantID: Balance
   */
  @example(#{ `01G65Z755AFWAKHE12NY0CQ9FH`: 100 })
  @visibility(Lifecycle.Read)
  grantBalancesAtStart: Record<float64>;

  /**
   * The entitlement balance at the end of the period.
   */
  @example(100)
  @visibility(Lifecycle.Read)
  balanceAtEnd: float64;

  /**
   * The balance breakdown of each active grant at the end of the period: GrantID: Balance
   */
  @example(#{ `01G65Z755AFWAKHE12NY0CQ9FH`: 100 })
  @visibility(Lifecycle.Read)
  grantBalancesAtEnd: Record<float64>;

  /**
   * Which grants were actually burnt down in the period and by what amount.
   */
  @visibility(Lifecycle.Read)
  grantUsages: GrantUsageRecord[];
}

/**
 * Usage Record
 */
@friendlyName("GrantUsageRecord")
model GrantUsageRecord {
  /**
   * The id of the grant
   */
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  grantId: ULID;

  /**
   * The usage in the period
   */
  @example(100)
  usage: float64;
}

/**
 * Reset parameters
 */
@friendlyName("ResetEntitlementUsageInput")
model ResetEntitlementUsageInput {
  /**
   * The time at which the reset takes effect, defaults to now. The reset cannot be in the future. The provided value is truncated to the minute due to how historical meter data is stored.
   */
  effectiveAt?: DateTime;

  /**
   * Determines whether the usage period anchor is retained or reset to the effectiveAt time.
   * - If true, the usage period anchor is retained.
   * - If false, the usage period anchor is reset to the effectiveAt time.
   */
  retainAnchor?: boolean;

  /**
   * Determines whether the overage is preserved or forgiven, overriding the entitlement's default behavior.
   * - If true, the overage is preserved.
   * - If false, the overage is forgiven.
   */
  preserveOverage?: boolean;
}
