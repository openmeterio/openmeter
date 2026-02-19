import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "../currency.tsp";
import "./cost-bases.tsp";
import "./cost-basis.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Currencies;

interface CostBasesOperations {
  /**
   * Get cost bases for a currency. For custom currencies, there can be multiple cost bases with different effective_from dates.
   * For standard currencies, there will be only one cost basis without an effective_from date.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-cost-bases")
  @summary("Get cost bases")
  get_cost_bases(
    @path currencyId: Shared.ULID,
  ): Shared.GetResponse<CostBases> | Common.ErrorResponses;

  /**
   * Create a cost basis for a currency. For custom currencies, there can be multiple cost bases with different effective_from dates.
   * For standard currencies, there will be only one cost basis without an effective_from date.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-cost-basis")
  @summary("Create cost basis")
  create_cost_basis(
    @header contentType: "application/json",
    @body body: CostBasis,
    @path currencyId: Shared.ULID,
  ): Shared.CreateResponse<CostBasis> | Common.ErrorResponses;
}
