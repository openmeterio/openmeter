import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "../customers/entitlements/entitlement.tsp";
import "../customers/entitlements/operations.tsp";
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
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.InternalExtension, true)
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

/**
 * Filter options for listing entitlements.
 */
@friendlyName("ListEntitlementsParamsFilter")
model ListEntitlementsParamsFilter {
  ...Customers.ListCustomerEntitlementsParamsFilter;

  /**
   * Filter entitlements by customer ID.
   */
  customer_id?: Common.ULIDFieldFilter;
}

interface EntitlementsOperations {
  /**
   * List the entitlements of every customer in the namespace that are active at the
   * time of the request. Intended for administrative use; to list the entitlements
   * of a single customer, use the customer entitlements endpoints, and for checking
   * entitlement access, use the entitlement access endpoints instead.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-entitlements")
  @summary("List entitlements")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort entitlements returned in the response. Supported sort attributes are:
     *
     * - `created_at` (default)
     * - `updated_at`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    /**
     * Filter entitlements returned in the response.
     *
     * To filter entitlements by customer add the following query param:
     * `filter[customer_id]=01K4WAQ0J99ZZ0MD75HXR112H8`
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListEntitlementsParamsFilter,
  ):
    | Shared.PagePaginatedResponse<Customers.Entitlement>
    | Common.ErrorResponses;

  /**
   * Get an entitlement by ID. For checking entitlement access, use the entitlement
   * access endpoints instead.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("get-entitlement")
  @summary("Get entitlement")
  get(@path entitlementId: Shared.ULID):
    | Shared.GetResponse<Customers.Entitlement>
    | Common.NotFound
    | Common.ErrorResponses;
}
