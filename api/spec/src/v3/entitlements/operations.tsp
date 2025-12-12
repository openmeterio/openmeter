import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../shared/index.tsp";
import "./access.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Entitlements;

/**
 * List the customer's active features and their access.
 */
@friendlyName("ListCustomerEntitlementAccessResponse")
model ListCustomerEntitlementAccessResponse {
  @Http.statusCode _: 200;
  @body body: ListCustomerEntitlementAccessResponseData;
}

/**
 * List customer entitlement access response data.
 */
@friendlyName("ListCustomerEntitlementAccessResponseData")
model ListCustomerEntitlementAccessResponseData {
  /**
   * The list of entitlement access results.
   */
  @visibility(Lifecycle.Read)
  data: EntitlementAccessResult[];
}

interface CustomerEntitlementsOperations {
  @get
  @operationId("list-customer-entitlement-access")
  @summary("List customer entitlement access")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  list(
    @path customerId: Shared.ULID,
  ): ListCustomerEntitlementAccessResponse | Common.NotFound | Common.ErrorResponses;
}
