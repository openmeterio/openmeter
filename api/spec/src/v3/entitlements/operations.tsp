import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/pagination.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./entitlement.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Entitlements;

interface CustomerEntitlementsOperations {
  @get
  @operationId("check-customer-feature-access")
  @summary("Check customer feature access")
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  get(
    @path customerId: Shared.ULID,
    @path featureKey: Shared.ResourceKey,
  ): Shared.GetResponse<EntitlementCheck> | Common.NotFound | Common.ErrorResponses;
}
