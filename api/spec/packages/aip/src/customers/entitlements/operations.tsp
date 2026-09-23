import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "./entitlement.tsp";
import "./history.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

/**
 * Filter options for listing customer entitlements.
 */
@friendlyName("ListCustomerEntitlementsParamsFilter")
model ListCustomerEntitlementsParamsFilter {
  /**
   * Filter entitlements by feature ID.
   */
  feature_id?: Common.ULIDFieldFilter;

  /**
   * Filter entitlements by feature key.
   */
  feature_key?: Common.StringFieldFilterExact;

  /**
   * Filter entitlements by type (`metered`, `static` or `boolean`).
   */
  type?: Common.StringFieldFilterExact;
}

namespace Entitlements {
  interface CustomerEntitlementsOperations {
    /**
     * Create an entitlement for the customer.
     *
     * A customer can have only one active entitlement per feature. The feature must be
     * compatible with the entitlement type. Entitlements cannot be modified after
     * creation, only deleted.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @post
    @operationId("create-customer-entitlement")
    @summary("Create customer entitlement")
    create(
      @path customerId: Shared.ULID,
      @body entitlement: CreateEntitlement,
    ):
      | Shared.CreateResponse<Entitlement>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * Override an entitlement of the customer.
     *
     * Overriding creates a new entitlement from the request and ends the referenced
     * entitlement at the same instant, so the customer keeps access without a gap. The
     * new entitlement must be for the same feature as the referenced one. Useful for
     * upgrades and downgrades.
     *
     * Fails if the referenced entitlement does not exist, is deleted, or is no longer
     * active.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @put
    @route("/{entitlementId}/override")
    @operationId("override-customer-entitlement")
    @summary("Override customer entitlement")
    override(
      @path customerId: Shared.ULID,
      @path entitlementId: Shared.ULID,
      @body entitlement: CreateEntitlement,
    ):
      | Shared.CreateResponse<Entitlement>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * Get the balance and usage history of a metered entitlement. The queried range
     * may span multiple usage periods.
     *
     * `windowed_history` groups usage into windows of the requested size and reports
     * the balance at the start of each window. `burndown_history` lists the periods in
     * which grants were consumed in a fixed order, together with the usage taken from
     * each grant.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.InternalExtension, true)
    @extension(Shared.PrivateExtension, true)
    @get
    @route("/{entitlementId}/history")
    @operationId("get-customer-entitlement-history")
    @summary("Get customer entitlement history")
    getHistory(
      @path customerId: Shared.ULID,
      @path entitlementId: Shared.ULID,

      /**
       * The start of the queried range. Defaults to the last reset. Truncated to the
       * minute. The range may span at most 1000 windows.
       */
      @query
      from?: Shared.DateTime,

      /**
       * The end of the queried range. Defaults to the current time. Truncated to the
       * minute unless it is the current time.
       */
      @query
      to?: Shared.DateTime,

      /**
       * The size of the windows the usage is grouped into.
       */
      @query
      window_size: EntitlementHistoryWindowSize,

      /**
       * The IANA time zone the windows are aligned to.
       */
      @query
      time_zone?: string = "UTC",
    ):
      | Shared.GetResponse<EntitlementHistory>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * Get an entitlement of the customer by ID. For checking entitlement access, use
     * the entitlement access endpoints instead.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @get
    @operationId("get-customer-entitlement")
    @summary("Get customer entitlement")
    get(@path customerId: Shared.ULID, @path entitlementId: Shared.ULID):
      | Shared.GetResponse<Entitlement>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * List the entitlements of the customer that are active at the time of the
     * request. For checking entitlement access, use the entitlement access endpoints
     * instead.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @get
    @operationId("list-customer-entitlements")
    @summary("List customer entitlements")
    list(
      @path customerId: Shared.ULID,
      ...Common.PagePaginationQuery,

      /**
       * Sort entitlements returned in the response. Supported sort attributes are:
       *
       * - `created_at` (default)
       * - `updated_at`
       *
       * The `asc` suffix is optional as the default sort order is ascending. The `desc`
       * suffix is used to specify a descending order.
       */
      @query(#{ name: "sort" })
      sort?: Common.SortQuery,

      /**
       * Filter entitlements returned in the response.
       *
       * To filter entitlements by feature key add the following query param:
       * `filter[feature_key]=my-feature`
       */
      @query(#{ style: "deepObject", explode: true })
      filter?: ListCustomerEntitlementsParamsFilter,
    ):
      | Shared.PagePaginatedResponse<Entitlement>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * Reset the usage of a metered entitlement. The reset starts a new usage period:
     * usage is zeroed and grants roll over according to their rollover settings.
     *
     * Usage is reset automatically at the end of each usage period. Use this operation
     * to reset it earlier, for example to align the entitlement with the customer's
     * billing period. The usage period anchor can be moved at the same time.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.InternalExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.RPCEndpointExtension, true)
    @post
    @route("/{entitlementId}/reset")
    @operationId("reset-customer-entitlement-usage")
    @summary("Reset customer entitlement usage")
    resetUsage(
      @path customerId: Shared.ULID,
      @path entitlementId: Shared.ULID,

      @body
      request?: ResetCustomerEntitlementUsageRequest,
    ):
      | Shared.DeleteResponse
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * Deletes the entitlement and revokes access to its feature. A customer can hold
     * only one active entitlement per feature, so migrating a feature requires
     * deleting the previous entitlement first.
     *
     * Deletion sets the `deleted_at` timestamp instead of removing history. Access and
     * status queries for earlier points in time still treat the entitlement as active,
     * so access changes are never retroactive.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @delete
    @operationId("delete-customer-entitlement")
    @summary("Delete customer entitlement")
    delete(@path customerId: Shared.ULID, @path entitlementId: Shared.ULID):
      | Shared.DeleteResponse
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;
  }
}
