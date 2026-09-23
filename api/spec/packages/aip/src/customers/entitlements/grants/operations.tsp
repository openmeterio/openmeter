import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../../common/error.tsp";
import "../../../common/parameters.tsp";
import "../../../common/pagination.tsp";
import "../../../shared/index.tsp";
import "../entitlement.tsp";
import "./grant.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

namespace Entitlements.Grants {
  interface CustomerEntitlementGrantsOperations {
    /**
     * Issue a grant for a metered entitlement of the customer. Grants can only be
     * issued for metered entitlements; the request is rejected for boolean and static
     * entitlements.
     *
     * Grants are immutable: the granted amount adds to the balance from
     * `effective_at`, which cannot be before the start of the current usage period.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @post
    @operationId("create-customer-entitlement-grant")
    @summary("Create customer entitlement grant")
    create(
      @path customerId: Shared.ULID,
      @path entitlementId: Shared.ULID,
      @body grant: EntitlementGrantCreate,
    ):
      | Shared.CreateResponse<EntitlementGrant>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * List the grants issued for an entitlement of the customer. Grants only exist for
     * metered entitlements, so the list of a boolean or static entitlement is empty.
     *
     * Deleted grants are excluded unless `include_deleted` is set. Voided and expired
     * grants are always included, as they are part of the balance history.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @get
    @operationId("list-customer-entitlement-grants")
    @summary("List customer entitlement grants")
    list(
      @path customerId: Shared.ULID,
      @path entitlementId: Shared.ULID,
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
       * Include deleted grants in the response.
       */
      @query
      include_deleted?: boolean,
    ):
      | Shared.PagePaginatedResponse<EntitlementGrant>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;
  }
}
