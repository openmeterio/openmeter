import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../shared/index.tsp";
import "./entitlement.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

namespace Entitlements {
  interface CustomerEntitlementsOperations {
    /**
     * OpenMeter has three types of entitlements: metered, boolean, and static. The
     * `type` property determines the type of entitlement. The underlying feature has
     * to be compatible with the entitlement type specified in the request (for
     * example, a metered entitlement needs a feature associated with a meter).
     *
     * - Boolean entitlements define static feature access, e.g. "Can use SSO
     *   authentication".
     * - Static entitlements let you pass along a configuration while granting access,
     *   e.g. "Using this feature with X Y settings" (passed in the config).
     * - Metered entitlements have many use cases, from setting up usage-based access
     *   to implementing complex credit systems. Example: the customer can use 10000 AI
     *   tokens during the usage period of the entitlement.
     *
     * A given customer can only have one active (non-deleted) entitlement per feature.
     * If you try to create a new entitlement for a feature that already has an active
     * entitlement, the request fails with a 409 error. If the customer has been
     * deleted, the request fails with a 412 error.
     *
     * Once an entitlement is created you cannot modify it, only delete it.
     */
    @extension(Shared.UnstableExtension, true)
    @post
    @operationId("create-customer-entitlement")
    @summary("Create customer entitlement")
    create(
      @path customerId: Shared.ULID,
      @body entitlement: CreateEntitlement,
    ):
      | Shared.CreateResponse<Entitlement>
      | Common.NotFound
      | Common.Conflict
      | Common.PreconditionFailed
      | Common.ErrorResponses;
  }
}
