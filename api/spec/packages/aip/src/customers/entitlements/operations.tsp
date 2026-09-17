import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

namespace Entitlements {
  interface CustomerEntitlementsOperations {
    /**
     * Deleting an entitlement revokes access to the associated feature. As a single
     * customer can only have one active entitlement per feature, when "migrating"
     * features you have to delete the old entitlements as well.
     *
     * As access and status checks can be historical queries, deleting an entitlement
     * populates the `deleted_at` timestamp. When queried for a time before that, the
     * entitlement is still considered active: you cannot have retroactive changes to
     * access, which is important for, among other things, auditing.
     */
    @extension(Shared.UnstableExtension, true)
    @delete
    @operationId("delete-customer-entitlement")
    @summary("Delete customer entitlement")
    delete(@path customerId: Shared.ULID, @path entitlementId: Shared.ULID):
      | Shared.DeleteResponse
      | Common.NotFound
      | Common.PreconditionFailed
      | Common.ErrorResponses;
  }
}
