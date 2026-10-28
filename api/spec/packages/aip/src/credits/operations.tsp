import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "../customers/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Credits;

/**
 * Filter options for listing credit grants across customers.
 */
@friendlyName("ListAllCreditGrantsParamsFilter")
model ListAllCreditGrantsParamsFilter {
  ...Customers.ListCreditGrantsParamsFilter;

  /**
   * Filter credit grants by the ID of their customer.
   */
  customer_id?: Common.ULIDFieldFilter;
}

/**
 * Filter options for listing credit balances across customers.
 */
@friendlyName("ListAllCreditBalancesParamsFilter")
model ListAllCreditBalancesParamsFilter {
  ...Customers.GetCreditBalanceParamsFilter;

  /**
   * Filter credit balances by the ID of their customer.
   */
  customer_id?: Common.ULIDFieldFilter;
}

/**
 * The credit balances of one customer.
 */
@friendlyName("BillingCustomerCreditBalances")
@summary("Customer credit balances")
model CustomerCreditBalances {
  /**
   * The customer owning the balances.
   */
  @visibility(Lifecycle.Read)
  customer: Shared.ResourceReference<Customers.Customer>;

  ...Customers.CreditBalances;
}

namespace Grants {
  interface CreditGrantsOperations {
    /**
     * List credit grants across all customers.
     *
     * Returns the same grants as the per-customer route; each grant carries its
     * `customer` reference.
     */
    @extension(Shared.UnstableExtension, true)
    @get
    @operationId("list-all-credit-grants")
    // Names the SDK method credits.grants.list and keeps the per-customer grant
    // listing's Go params struct (CreditGrantListParams) unchanged.
    @friendlyName("listCreditsGrants")
    @summary("List credit grants across customers")
    list(
      ...Common.PagePaginationQuery,

      /**
       * Sort credit grants by `created_at`, `updated_at`, `name`, or `id`. Defaults to
       * `created_at desc`. The `asc` suffix is optional when a sort attribute is
       * supplied. Supports one attribute; ties are broken by `id` in the same direction.
       */
      @query(#{ name: "sort" })
      sort?: Common.SortQuery,

      /**
       * Filter credit grants returned in the response.
       *
       * To filter by customer use `filter[customer_id][eq]=<id>` or
       * `filter[customer_id][oeq]=<id>,<id>`.
       */
      @query(#{ style: "deepObject", explode: true })
      filter?: ListAllCreditGrantsParamsFilter,
    ):
      | Shared.PagePaginatedResponse<Customers.CreditGrant>
      | Common.ErrorResponses;
  }
}

namespace Balances {
  interface CreditBalancesOperations {
    /**
     * List credit balances across customers.
     *
     * Returns one item per customer, ordered by customer ID. Customers without credit
     * activity are returned with an empty `balances` list. The page size is limited to
     * 100 because balances are computed on request.
     */
    @extension(Shared.UnstableExtension, true)
    @get
    @operationId("list-all-credit-balances")
    // Names the SDK method credits.balances.list.
    @friendlyName("listCreditsBalances")
    @summary("List credit balances across customers")
    list(
      ...Common.PagePaginationQuery,

      /**
       * Return credit balances at this timestamp using all currently recorded postings,
       * including corrections recorded later. This is not a snapshot of what was known
       * at the requested time.
       *
       * Defaults to the current time. Historical responses return `live` as zero because
       * live charge impacts are only available for current balances. A historical read
       * can be re-run for a daily or month-end close; it returns the same result until a
       * correction is booked with an earlier timestamp.
       */
      @query
      timestamp?: Shared.DateTime,

      /**
       * Filter credit balances returned in the response.
       *
       * To filter by customer use `filter[customer_id][eq]=<id>` or
       * `filter[customer_id][oeq]=<id>,<id>`.
       */
      @query(#{ style: "deepObject", explode: true })
      filter?: ListAllCreditBalancesParamsFilter,
    ):
      | Shared.PagePaginatedResponse<CustomerCreditBalances>
      | Common.ErrorResponses;
  }
}
