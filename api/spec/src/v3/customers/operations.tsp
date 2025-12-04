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

interface CustomersOperations {
  @post
  @operationId("create-customer")
  @summary("Create customer")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  create(
    @body
    customer: Shared.CreateRequest<Customer>,
  ): Shared.CreateResponse<Customer> | Common.ErrorResponses;

  @get
  @operationId("get-customer")
  @summary("Get customer")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  get(
    @path customerId: Shared.ULID,
  ): Shared.GetResponse<Customer> | Common.NotFound | Common.ErrorResponses;

  @get
  @operationId("list-customers")
  @summary("List customers")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  list(
    ...Common.CursorPageQuery,
  ): Shared.CursorPaginatedResponse<Customer> | Common.ErrorResponses;
}
