import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../common/pagination.tsp";
import "../shared/index.tsp";
import "../customers/entitlements/grants/grant.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Grants;

/**
 * Filter options for listing grants.
 */
@friendlyName("ListGrantsParamsFilter")
model ListGrantsParamsFilter {
  /**
   * Filter grants by the ID of the customer that owns the entitlement. Only the `eq`
   * and `oeq` operators are supported.
   */
  customer_id?: Common.ULIDFieldFilter;

  /**
   * Filter grants by the ID or the key of the entitlement's feature. Only the `eq`
   * and `oeq` operators are supported.
   */
  feature?: Common.StringFieldFilterExact;
}

interface GrantsOperations {
  /**
   * List the grants of all customers and entitlements. To list the grants of a
   * single entitlement, use the customer entitlement grants endpoint.
   *
   * Deleted grants are excluded unless `include_deleted` is set. Voided and expired
   * grants are always included, as they are part of the balance history.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-grants")
  @summary("List grants")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort grants returned in the response. Supported sort attributes are:
     *
     * - `created_at` (default)
     * - `updated_at`
     * - `effective_at`
     * - `expires_at`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter grants returned in the response.
     *
     * To filter grants by customer, add the following query param:
     * `filter[customer_id]=01K4WAQ0J99ZZ0MD75HXR112H8`
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListGrantsParamsFilter,

    /**
     * Include deleted grants in the response.
     */
    @query
    include_deleted?: boolean,
  ):
    | Shared.PagePaginatedResponse<Customers.EntitlementGrant>
    | Common.ErrorResponses;

  /**
   * Void a grant so it no longer adds to the balance. Usage already deducted from
   * the grant is kept.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.InternalExtension, true)
  @delete
  @operationId("void-grant")
  @summary("Void grant")
  delete(
    @path grantId: Shared.ULID,

    /**
     * The time the grant is voided from. Defaults to now; it cannot be in the future
     * or before the start of the current usage period.
     */
    @query
    voided_at?: Shared.DateTime,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}
