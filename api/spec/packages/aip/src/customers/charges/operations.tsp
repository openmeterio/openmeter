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
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "feature_id and feature_key should not be grouped"
@friendlyName("ListChargesParamsFilter")
model ListCustomerChargesParamsFilter {
  /**
   * Filter charges by status.
   *
   * Supported statuses are:
   *
   * - `created`
   * - `active`
   * - `final`
   * - `deleted`
   *
   * If omitted, all statuses are returned except for `deleted`.
   */
  status?: Common.StringFieldFilterExact;

  /**
   * Filter charges by the ID of their associated feature.
   */
  feature_id?: Common.ULIDFieldFilter;

  /**
   * Filter charges by the key of their associated feature.
   */
  feature_key?: Common.StringFieldFilterExact;

  /**
   * Filter charges by the start of their service period.
   *
   * Combine with `service_period_to` to match charges whose service period falls
   * within a given window: `filter[service_period_from][gte]=<from>` together with
   * `filter[service_period_to][lte]=<to>` returns charges whose service period lies
   * within `[from, to)`.
   */
  service_period_from?: Common.DateTimeFieldFilter;

  /**
   * Filter charges by the end of their service period.
   *
   * See `service_period_from` for how to express a service-period window query using
   * both fields together.
   */
  service_period_to?: Common.DateTimeFieldFilter;
}

namespace Charges {
  interface CustomerChargesOperations {
    /**
     * List customer charges.
     *
     * Returns the customer's charges that are represented as either flat fee or
     * usage-based charges.
     */
    @extension(Shared.UnstableExtension, true)
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
      filter?: ListCustomerChargesParamsFilter,

      /**
       * Expand full objects for referenced entities.
       *
       * See `BillingChargesExpand` for the supported values and what each one populates.
       */
      @query(#{ explode: false })
      expand?: ChargesExpand[],
    ):
      | Shared.PagePaginatedResponse<Charge>
      | Common.NotFound
      | Common.ErrorResponses;

    /**
     * Create customer charge.
     */
    @extension(Shared.UnstableExtension, true)
    @post
    @operationId("create-customer-charges")
    @summary("Create customer charge")
    create(@path customerId: Shared.ULID, @body charge: CreateCharge):
      | Shared.CreateResponse<Charge>
      | Common.ErrorResponses;
  }
}
