import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "..";
import "../..";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v2/customers/{customerIdOrKey}/entitlements")
@tag("Entitlements")
@friendlyName("CustomerEntitlementsV2")
interface CustomerEntitlementsV2Endpoints {
  /**
   * OpenMeter has three types of entitlements: metered, boolean, and static. The type property determines the type of entitlement. The underlying feature has to be compatible with the entitlement type specified in the request (e.g., a metered entitlement needs a feature associated with a meter).
   *
   * - Boolean entitlements define static feature access, e.g. "Can use SSO authentication".
   * - Static entitlements let you pass along a configuration while granting access, e.g. "Using this feature with X Y settings" (passed in the config).
   * - Metered entitlements have many use cases, from setting up usage-based access to implementing complex credit systems.  Example: The customer can use 10000 AI tokens during the usage period of the entitlement.
   *
   * A given customer can only have one active (non-deleted) entitlement per featureKey. If you try to create a new entitlement for a featureKey that already has an active entitlement, the request will fail with a 409 error.
   *
   * Once an entitlement is created you cannot modify it, only delete it.
   */
  @tag("Customers")
  @post
  @summary("Create a customer entitlement")
  @operationId("createCustomerEntitlementV2")
  post(
    @path customerIdOrKey: ULIDOrExternalKey,
    @body entitlement: EntitlementV2CreateInputs,
  ): {
    @statusCode _: 201;
    @body body: EntitlementV2;
  } | CommonErrors | ConflictError;

  /**
   * List all entitlements for a customer. For checking entitlement access, use the /value endpoint instead.
   */
  @get
  @operationId("listCustomerEntitlementsV2")
  @summary("List customer entitlements")
  list(
    @path customerIdOrKey: ULIDOrExternalKey,
    @query includeDeleted?: boolean = false,
    ...QueryPagination,
    ...QueryOrdering<EntitlementOrderBy>,
  ): PaginatedResponse<EntitlementV2> | CommonErrors;

  /**
   * Get entitlement by feature key. For checking entitlement access, use the /value endpoint instead.
   * If featureKey is used, the entitlement is resolved for the current timestamp.
   */
  @get
  @operationId("getCustomerEntitlementV2")
  @summary("Get customer entitlement")
  @route("/{entitlementIdOrFeatureKey}")
  get(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrKey,
  ): EntitlementV2 | CommonErrors | NotFoundError;

  /**
   * Deleting an entitlement revokes access to the associated feature. As a single customer can only have one entitlement per featureKey, when "migrating" features you have to delete the old entitlements as well.
   * As access and status checks can be historical queries, deleting an entitlement populates the deletedAt timestamp. When queried for a time before that, the entitlement is still considered active, you cannot have retroactive changes to access, which is important for, among other things, auditing.
   */
  @delete
  @operationId("deleteCustomerEntitlementV2")
  @summary("Delete customer entitlement")
  @route("/{entitlementIdOrFeatureKey}")
  delete(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrKey,
  ): {
    @statusCode _: 204;
  } | CommonErrors | NotFoundError;

  /**
   * Overriding an entitlement creates a new entitlement from the provided inputs and soft deletes the previous entitlement for the provided customer-feature pair. If the previous entitlement is already deleted or otherwise doesnt exist, the override will fail.
   *
   * This endpoint is useful for upgrades, downgrades, or other changes to entitlements that require a new entitlement to be created with zero downtime.
   */
  @put
  @operationId("overrideCustomerEntitlementV2")
  @route("/{entitlementIdOrFeatureKey}/override")
  @summary("Override customer entitlement")
  override(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrExternalKey,
    @body entitlement: EntitlementV2CreateInputs,
  ):
    | {
        @statusCode _: 201;
        @body body: EntitlementV2;
      }
    | CommonErrors
    | ConflictError
    | NotFoundError;
}

@route("/api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}")
@tag("Entitlements")
@tag("Customers")
@friendlyName("CustomerEntitlementV2")
interface CustomerEntitlementV2Endpoints {
  /**
   * List all grants issued for an entitlement. The entitlement can be defined either by its id or featureKey.
   */
  @get
  @operationId("listCustomerEntitlementGrantsV2")
  @route("/grants")
  @summary("List customer entitlement grants")
  getGrants(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrKey,
    @query includeDeleted?: boolean = false,
    ...QueryPagination,
    ...QueryLimitOffset,
    ...QueryOrdering<GrantOrderBy>,
  ): PaginatedResponse<GrantV2> | CommonErrors;

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
  @operationId("createCustomerEntitlementGrantV2")
  @route("/grants")
  @summary("Create customer entitlement grant")
  createCustomerEntitlementGrant(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrKey,
    @body grant: GrantCreateInputV2,
  ): {
    @statusCode _: 201;
    @body body: GrantV2;
  } | CommonErrors | ConflictError;

  /**
   * Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
   */
  @get
  @operationId("getCustomerEntitlementValueV2")
  @route("/value")
  @summary("Get customer entitlement value")
  getCustomerEntitlementValue(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrKey,
    @query(#{ explode: true }) time?: DateTime,
  ): EntitlementValue | CommonErrors | NotFoundError;

  /**
   * Returns historical balance and usage data for the entitlement. The queried history can span accross multiple reset events.
   *
   * BurndownHistory returns a continous history of segments, where the segments are seperated by events that changed either the grant burndown priority or the usage period.
   *
   * WindowedHistory returns windowed usage data for the period enriched with balance information and the list of grants that were being burnt down in that window.
   */
  @get
  @operationId("getCustomerEntitlementHistoryV2")
  @route("/history")
  @summary("Get customer entitlement history")
  getCustomerEntitlementHistory(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrKey,

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
   * Reset marks the start of a new usage period for the entitlement and initiates grant rollover. At the start of a period usage is zerod out and grants are rolled over based on their rollover settings. It would typically be synced with the customers billing period to enforce usage based on their subscription.
   *
   * Usage is automatically reset for metered entitlements based on their usage period, but this endpoint allows to manually reset it at any time. When doing so the period anchor of the entitlement can be changed if needed.
   */
  @post
  @operationId("resetCustomerEntitlementUsageV2")
  @route("/reset")
  @summary("Reset customer entitlement")
  resetCustomerEntitlement(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path entitlementIdOrFeatureKey: ULIDOrKey,
    @body reset: ResetEntitlementUsageInput,
  ): {
    @statusCode _: 204;
  } | CommonErrors | NotFoundError;
}
