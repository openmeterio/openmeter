import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../shared/index.tsp";
import "./taxconfig.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Organization;

interface OrgTaxConfigOperations {
  @get
  @operationId("get-org-tax-config")
  @summary("Get organization tax config")
  get(
  ): Shared.GetResponse<OrgTaxConfig> | Common.NotFound | Common.ErrorResponses;

  @put
  @operationId("update-org-tax-config")
  @summary("Update organization tax config")
  upsert(
    @body body: OrgTaxConfigUpdate,
  ): Shared.UpsertResponse<OrgTaxConfig> | Common.NotFound | Common.ErrorResponses;
}
