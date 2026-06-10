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
 * Customer charge.
 */
@friendlyName("CreateChargeRequest")
@summary("Customer charge create request.")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union CreateCharge {
  /**
   * A flat fee charge.
   */
  flat_fee: Shared.CreateRequest<FlatFeeCharge>,

  /**
   * A usage-based charge.
   */
  usage_based: Shared.CreateRequest<UsageBasedCharge>,
}

/**
 * Filter options for listing charges.
 */
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
       */
      @query(#{ style: "deepObject", explode: true })
      filter?: ListCustomerChargesParamsFilter,

      /**
       * Expand full objects for referenced entities.
       *
       * Supported values are:
       *
       * - `real_time_usage`: Expand the charge's real-time usage.
       */
      @query(#{ name: "expand", explode: true })
      expand?: ChargesExpand[],
    ):
      | Shared.PagePaginatedResponse<Charge>
      | Common.NotFound
      | Common.ErrorResponses;

    /**
     * Create customer charge.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.InternalExtension, true)
    @post
    @operationId("create-customer-charges")
    @summary("Create customer charge")
    create(@path customerId: Shared.ULID, @body charge: CreateCharge):
      | Shared.CreateResponse<Charge>
      | Common.ErrorResponses;
  }
}
