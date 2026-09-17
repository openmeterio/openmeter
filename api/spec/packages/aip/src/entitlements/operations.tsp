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
  list(@path customerId: Shared.ULID):
    | ListCustomerEntitlementAccessResponse
    | Common.NotFound
    | Common.ErrorResponses;

  /**
   * Get the customer's access to a single feature.
   */
  @get
  @route("/features/{featureKey}")
  @operationId("get-customer-entitlement-access")
  @summary("Get customer entitlement access")
  get(
    @path customerId: Shared.ULID,
    @path featureKey: Shared.ResourceKey,

    /**
     * Expand computed fields.
     *
     * Supported values are:
     *
     * - `value`: Expand the balance details of a metered entitlement; it sets the
     *   `value` field.
     */
    @query(#{ explode: false })
    expand?: EntitlementAccessExpand[],
  ):
    | Shared.GetResponse<EntitlementAccessResult>
    | Common.NotFound
    | Common.ErrorResponses;
}
