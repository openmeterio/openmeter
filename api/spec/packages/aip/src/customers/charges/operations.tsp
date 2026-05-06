import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/pagination.tsp";
import "../../shared/index.tsp";
import "../../common/parameters.tsp";
import "./charges.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

/**
 * Filter options for listing charges.
 */
@friendlyName("ListChargesParamsFilter")
model ListCustomerChargesParamsFilter {
  /**
   * Filter charges by status.
   *
   * Supported statuses are:
   * - `created`
   * - `active`
   * - `final`
   * - `deleted`
   *
   * If omitted, all statuses are returned except for `deleted`.
   */
  status?: Common.StringFieldFilterExact;
}

interface CustomerChargesOperations {
  /**
   * List customer charges.
   *
   * Returns the customer's charges that are represented as either flat fee or usage-based charges.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-customer-charges")
  @summary("List customer charges")
  list(
    ...Common.PagePaginationQuery,
    @path customerId: Shared.ULID,

    /**
     * Sort charges returned in the response.
     *
     * Supported sort attributes are:
     * - `id`
     * - `created_at`
     * - `service_period.from`
     * - `billing_period.from`
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter charges.
     *
     * To filter charges by status add the following query param: `filter[status][oeq]=created,active`
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListCustomerChargesParamsFilter,

    /**
     * Expand full objects for referenced entities.
     *
     * Supported values are:
     * - `real_time_usage`: Expand the charge's real-time usage.
     */
    @query(#{ name: "expand", explode: true })
    expand?: ChargesExpand[],
  ): Shared.PagePaginatedResponse<Charge> | Common.NotFound | Common.ErrorResponses;
}
