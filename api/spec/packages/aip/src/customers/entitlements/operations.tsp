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
     * Create an entitlement for the customer.
     *
     * A customer can have only one active entitlement per feature. The feature must be
     * compatible with the entitlement type. Entitlements cannot be modified after
     * creation, only deleted.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
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
      | Common.ErrorResponses;
  }
}
