import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./customer.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

/**
 * Filter options for listing customers.
 */
@friendlyName("ListCustomersParamsFilter")
model ListCustomersParamsFilter {
  /**
   * Filter customers by key.
   * Case-insensitive partial match.
   */
  key?: Shared.ExternalResourceKey;
}

interface CustomersOperations {
  @post
  @operationId("create-customer")
  @summary("Create customer")
  create(
    @body
    customer: Shared.CreateRequest<Customer>,
  ): Shared.CreateResponse<Customer> | Common.ErrorResponses;

  @get
  @operationId("get-customer")
  @summary("Get customer")
  get(
    @path customerId: Shared.ULID,
  ): Shared.GetResponse<Customer> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("list-customers")
  @summary("List customers")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort customers returned in the response.
     * Supported sort attributes are:
     * - `id`
     * - `name` (default)
     * - `created_at`
     *
     * The `asc` suffix is optional as the default sort order is ascending.
     * The `desc` suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter customers returned in the response.
     *
     * To filter customers by key add the following query param: filter[key]=my-db-id
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListCustomersParamsFilter,
  ): Shared.PagePaginatedResponse<Customer> | Common.ErrorResponses;

  @put
  @operationId("upsert-customer")
  @summary("Upsert customer")
  upsert(
    @path customerId: Shared.ULID,

    @body
    customer: Shared.UpsertRequest<Customer>,
  ):
    | Shared.UpsertResponse<Customer>
    | Common.Gone
    | Common.NotFound
    | Common.ErrorResponses;

  @delete
  @operationId("delete-customer")
  @summary("Delete customer")
  delete(
    @path customerId: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}
