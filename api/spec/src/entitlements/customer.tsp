import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Entitlements;

@route("/api/v1/customers/{customerIdOrKey}")
@tag("Entitlements")
@tag("Customers")
interface Customer {
  /**
   * Checks customer access to a given feature (by key). All entitlement types share the hasAccess property in their value response, but multiple other properties are returned based on the entitlement type.
   */
  @get
  @operationId("getCustomerEntitlementValue")
  @route("/entitlements/{featureKey}/value")
  @summary("Get customer entitlement value")
  getCustomerEntitlementValue(
    @path customerIdOrKey: ULIDOrExternalKey,
    @path featureKey: Key,
    @query(#{ explode: true }) time?: DateTime,
  ): EntitlementValue | OpenMeter.CommonErrors | OpenMeter.NotFoundError;

  /**
   * Get the overall access of a customer.
   */
  @get
  @operationId("getCustomerAccess")
  @route("/access")
  @summary("Get customer access")
  getCustomerAccess(
    @path customerIdOrKey: ULIDOrExternalKey,
  ): CustomerAccess | OpenMeter.CommonErrors | OpenMeter.NotFoundError;
}

/**
 * CustomerAccess describes what features the customer has access to.
 */
@friendlyName("CustomerAccess")
model CustomerAccess {
  /**
   * Map of entitlements the customer has access to.
   * The key is the feature key, the value is the entitlement value + the entitlement ID.
   */
  @visibility(Lifecycle.Read)
  entitlements: Record<EntitlementValue>;
}
