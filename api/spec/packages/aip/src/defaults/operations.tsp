import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../shared/index.tsp";
import "./taxcodes.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Defaults;

interface OrganizationDefaultTaxCodesOperations {
  @get
  @operationId("get-organization-default-tax-codes")
  @summary("Get organization default tax codes")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  get(
  ): Shared.GetResponse<OrgDefaultTaxCodes> | Common.NotFound | Common.ErrorResponses;

  @put
  @operationId("update-organization-default-tax-codes")
  @summary("Update organization default tax codes")
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  update(
    @body body: OrgDefaultTaxCodes,
  ): Shared.UpsertResponse<OrgDefaultTaxCodes> | Common.NotFound | Common.ErrorResponses;
}
