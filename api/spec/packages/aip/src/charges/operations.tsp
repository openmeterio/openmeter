import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "../customers/charges/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Charges;

/**
 * Filter options for listing charges across customers.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "feature_id and feature_key should not be grouped"
@friendlyName("ListChargesParamsFilter")
model ListChargesParamsFilter {
  ...Customers.ListCustomerChargesParamsFilter;

  /**
   * Filter charges by the ID of their customer.
   */
  customer_id?: Common.ULIDFieldFilter;
}

interface ChargesOperations {
  /**
   * List charges.
   *
   * Returns the charges of every customer that are represented as either flat fee or
   * usage-based charges.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-charges")
  @summary("List charges")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort charges returned in the response.
     *
     * Supported sort attributes are:
     *
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
     * To filter charges by customer add the following query param:
     * `filter[customer_id][eq]=<id>` or `filter[customer_id][oeq]=<id>,<id>`.
     *
     * To filter charges by status add the following query param:
     * `filter[status][oeq]=created,active`
     *
     * To filter charges by feature, use `filter[feature_id][oeq]=<id>,<id>` or
     * `filter[feature_key][oeq]=<key>,<key>`.
     *
     * See the `service_period_from` filter field for expressing a service-period
     * window query.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListChargesParamsFilter,

    /**
     * Expand full objects for referenced entities.
     *
     * Supported values are:
     *
     * - `real_time_usage`: Expand the charge's real-time usage; it sets the `usage`
     *   and the `totals.realtime` fields.
     * - `customer`: Expand the charge's customer to the complete entity.
     * - `feature`: Expand the charge's feature to the complete entity.
     * - `subscription`: Expand the charge's subscription to the complete entity.
     * - `realization.invoice`: Expand each realization's invoice to the complete
     *   entity.
     * - `realization.totals`: Expand each realization run's `totals`.
     * - `realization.detailed_lines`: Expand each realization run's `detailed_lines`.
     */
    @query(#{ explode: false })
    expand?: Customers.ChargesExpand[],
  ): Shared.PagePaginatedResponse<Customers.Charge> | Common.ErrorResponses;
}
