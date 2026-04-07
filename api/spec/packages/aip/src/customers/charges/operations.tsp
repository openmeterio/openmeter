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

interface CustomerChargesOperations {
  /**
   * List customer charges.
   *
   * Returns the customer's charges that are represented as either flat fee or usage-based charges.
   */
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
     * Filter charges by status.
     *
     * Supported statuses are:
     * - `created`
     * - `active`
     * - `settled`
     * - `final`
     * - `deleted`
     *
     * If omitted, all statuses are returned except for `deleted`.
     */
    @query(#{ name: "status", explode: true })
    status?: ChargeStatus[],

    /**
     * Expands
     */
    @query(#{ name: "expand", explode: true })
    expand?: ChargesExpand[],
  ): Shared.PagePaginatedResponse<Charge> | Common.NotFound | Common.ErrorResponses;
}
