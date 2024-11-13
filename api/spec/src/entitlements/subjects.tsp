import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

// TODO: does this have to be in a separate namespace?
namespace OpenMeter.Entitlements;

@route("/api/v1/subjects/{subjectIdOrKey}/entitlements")
@tag("Entitlements (Beta)")
interface Subjects {
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
   */
  @post
  @summary("Create an entitlement")
  @operationId("createEntitlement")
  post(
    @path subjectIdOrKey: string,
    @body entitlement: EntitlementCreateInputs,
  ): {
    @statusCode _: 201;
    @body body: Entitlement;
  } | OpenMeter.CommonErrors | OpenMeter.ConflictError;

  /**
   * List all entitlements for a subject. For checking entitlement access, use the /value endpoint instead.
   */
  @get
  @operationId("listSubjectEntitlements")
  @summary("List entitlements")
  list(
    @path subjectIdOrKey: string,
    @query includeDeleted?: boolean = false,
  ): Entitlement[] | OpenMeter.CommonErrors;

  /**
   * Get entitlement by id. For checking entitlement access, use the /value endpoint instead.
   */
  @get
  @operationId("getEntitlement")
  @summary("Get entitlement")
  get(
    @path subjectIdOrKey: string,
    @path entitlementId: string,
  ): Entitlement | OpenMeter.CommonErrors | OpenMeter.NotFoundError;

  /**
   * Deleting an entitlement revokes access to the associated feature. As a single subject can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well.
   * As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.
   */
  @delete
  @operationId("deleteEntitlement")
  @summary("Delete entitlement")
  delete(@path subjectIdOrKey: string, @path entitlementId: string): {
    @statusCode _: 204;
  } | OpenMeter.CommonErrors | OpenMeter.NotFoundError;

  /**
   * Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided subject-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.
   *
   * This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.
   */
  @put
  @operationId("overrideEntitlement")
  @route("/{entitlementIdOrFeatureKey}/override")
  @summary("Override entitlement")
  override(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @body entitlement: EntitlementCreateInputs,
  ):
    | {
        @statusCode _: 201;
        @body body: Entitlement;
      }
    | OpenMeter.CommonErrors
    | OpenMeter.ConflictError
    | OpenMeter.NotFoundError;

  /**
   * List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.
   */
  @get
  @operationId("listEntitlementGrants")
  @route("/{entitlementIdOrFeatureKey}/grants")
  @summary("List entitlement grants")
  getGrants(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @query includeDeleted?: boolean = false,
    @query orderBy?: GrantOrderBy = GrantOrderBy.UpdatedAt,
  ): Grant[] | OpenMeter.CommonErrors;

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
   */
  @post
  @operationId("createGrant")
  @route("/{entitlementIdOrFeatureKey}/grants")
  @summary("Create grant")
  createGrant(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @body grant: GrantCreateInput,
  ): {
    @statusCode _: 201;
    @body body: Grant;
  } | OpenMeter.CommonErrors | OpenMeter.ConflictError;

  /**
   * This endpoint should be used for access checks and enforcement. All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
   *
   * For convenience reasons, /value works with both entitlementId and featureKey.
   */
  @get
  @operationId("getEntitlementValue")
  @route("/{entitlementIdOrFeatureKey}/value")
  @summary("Get entitlement value")
  getEntitlementValue(
    @path subjectIdOrKey: string,
    @path entitlementIdOrFeatureKey: string,
    @query(#{ explode: true }) time?: DateTime,
  ): EntitlementValue | OpenMeter.CommonErrors | OpenMeter.NotFoundError;

  /**
   * Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.
   *
   * BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.
   *
   * WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.
   */
  @get
  @operationId("getEntitlementHistory")
  @route("/{entitlementId}/history")
  @summary("Get entitlement history")
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
    @query(#{ explode: true }) windowSize: OpenMeter.WindowSize,

    /**
     * The timezone used when calculating the windows.
     */
    @query(#{ explode: true }) windowTimeZone?: string = "UTC",
  ): WindowedBalanceHistory | OpenMeter.CommonErrors | OpenMeter.NotFoundError;

  /**
   * Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the subjects billing period to enforce usage based on their subscription.
   *
   * Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.
   */
  @post
  @operationId("resetEntitlementUsage")
  @route("/{entitlementId}/reset")
  @summary("Reset entitlement")
  reset(
    @path subjectIdOrKey: string,
    @path entitlementId: string,
    @body reset: ResetEntitlementUsageInput,
  ): {
    @statusCode _: 204;
  } | OpenMeter.CommonErrors | OpenMeter.NotFoundError;
}

/**
 * Entitlements are the core of OpenMeter access management. They define access to features for subjects. Entitlements can be metered, boolean, or static.
 */
@friendlyName("EntitlementValue")
model EntitlementValue {
  /**
   * Whether the subject has access to the feature. Shared accross all entitlement types.
   */
  @visibility("read")
  @example(true)
  hasAccess: boolean;

  /**
   * Only available for metered entitlements. Metered entitlements are built around a balance calculation where feature usage is deducted from the issued grants. Balance represents the remaining balance of the entitlement, it's value never turns negative.
   */
  @example(100)
  @visibility("read")
  balance?: float64;

  /**
   * Only available for metered entitlements. Returns the total feature usage in the current period.
   */
  @example(50)
  @visibility("read")
  usage?: float64;

  /**
   * Only available for metered entitlements. Overage represents the usage that wasn't covered by grants, e.g. if the subject had a total feature usage of 100 in the period but they were only granted 80, there would be 20 overage.
   */
  @example(0)
  @visibility("read")
  overage?: float64;

  /**
   * Only available for static entitlements. The JSON parsable config of the entitlement.
   */
  @example("{ key: \"value\" }")
  @visibility("read")
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
  @visibility("read")
  usage: float64;

  /**
   * The entitlement balance at the start of the period.
   */
  @example(100)
  @visibility("read")
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
  @visibility("read")
  usage: float64;

  /**
   * Overuse that wasn't covered by grants.
   */
  @example(100)
  @visibility("read")
  overage: float64;

  /**
   * entitlement balance at the start of the period.
   */
  @example(100)
  @visibility("read")
  balanceAtStart: float64;

  /**
   * The balance breakdown of each active grant at the start of the period: GrantID: Balance
   */
  @example(#{ `01G65Z755AFWAKHE12NY0CQ9FH`: 100 })
  @visibility("read")
  grantBalancesAtStart: Record<float64>;

  /**
   * The entitlement balance at the end of the period.
   */
  @example(100)
  @visibility("read")
  balanceAtEnd: float64;

  /**
   * The balance breakdown of each active grant at the end of the period: GrantID: Balance
   */
  @example(#{ `01G65Z755AFWAKHE12NY0CQ9FH`: 100 })
  @visibility("read")
  grantBalancesAtEnd: Record<float64>;

  /**
   * Which grants were actually burnt down in the period and by what amount.
   */
  @visibility("read")
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
