import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "..";
import "../..";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v2/grants")
@tag("Entitlements")
@friendlyName("GrantsV2")
interface GrantsV2Endpoints {
  /**
   * List all grants for all the customers and entitlements. This endpoint is intended for administrative purposes only.
   * To fetch the grants of a specific entitlement please use the /api/v2/customers/{customerIdOrKey}/entitlements/{entitlementIdOrFeatureKey}/grants endpoint.
   * If page is provided that takes precedence and the paginated response is returned.
   */
  @get
  @operationId("listGrantsV2")
  @summary("List grants")
  list(
    /**
     * Filtering by multiple features.
     *
     * Usage: `?feature=feature-1&feature=feature-2`
     */
    @query(#{ explode: true })
    feature?: string[],

    /**
     * Filtering by multiple customers (either by ID or key).
     *
     * Usage: `?customer=customer-1&customer=customer-2`
     */
    @query(#{ explode: true })
    customer?: ULIDOrExternalKey[],

    /**
     * Include deleted
     */
    @query includeDeleted?: boolean = false,

    ...QueryPagination,
    ...QueryLimitOffset,
    ...QueryOrdering<GrantOrderBy>,
  ): PaginatedResponse<GrantV2> | CommonErrors;
}
