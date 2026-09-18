import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "./entitlement.tsp";

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
     * OpenMeter has three types of entitlements: metered, boolean, and static. The
     * `type` property determines the type of entitlement. The underlying feature has
     * to be compatible with the entitlement type specified in the request (for
     * example, a metered entitlement needs a feature associated with a meter).
     *
     * - Boolean entitlements define static feature access, e.g. "Can use SSO
     *   authentication".
     * - Static entitlements let you pass along a configuration while granting access,
     *   e.g. "Using this feature with X Y settings" (passed in the config).
     * - Metered entitlements have many use cases, from setting up usage-based access
     *   to implementing complex credit systems. Example: the customer can use 10000 AI
     *   tokens during the usage period of the entitlement.
     *
     * A given customer can only have one active (non-deleted) entitlement per feature.
     * If you try to create a new entitlement for a feature that already has an active
     * entitlement, the request fails with a 409 error.
     *
     * Once an entitlement is created you cannot modify it, only delete it.
     */
    @extension(Shared.UnstableExtension, true)
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
      | Common.PreconditionFailed
      | Common.ErrorResponses;

    /**
     * Get an entitlement of the customer by ID. For checking entitlement access, use
     * the entitlement access endpoints instead.
     */
    @extension(Shared.UnstableExtension, true)
    @get
    @operationId("get-customer-entitlement")
    @summary("Get customer entitlement")
    get(@path customerId: Shared.ULID, @path entitlementId: Shared.ULID):
      | Shared.GetResponse<Entitlement>
      | Common.NotFound
      | Common.PreconditionFailed
      | Common.ErrorResponses;

    /**
     * List the entitlements of the customer that are active at the time of the
     * request. For checking entitlement access, use the entitlement access endpoints
     * instead.
     */
    @extension(Shared.UnstableExtension, true)
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
      | Common.PreconditionFailed
      | Common.ErrorResponses;
  }
}
