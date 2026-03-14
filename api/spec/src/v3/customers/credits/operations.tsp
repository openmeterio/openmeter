import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "./credit.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

interface CustomerCreditsOperations {
  /**
   * Create a new credit grant.
   * A credit grant represents an allocation of prepaid credits to a customer.
   */
  @post
  @operationId("create-credit-grant")
  @summary("Create a new credit grant")
  create(
    @path customerId: Shared.ULID,
    @body credit_grant: Shared.CreateRequest<CreditGrant>,
  ): Shared.CreateResponse<CreditGrant> | Common.ErrorResponses;
}
