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
   * Get cost bases by currency id
   */
  @get
  @operationId("get-cost-bases-by-id")
  @summary("Get cost bases by id")
  get_cost_bases(
    @path currencyId: Shared.ULID,
  ): CostBases | Common.ErrorResponses;

  /**
   * Create cost basis for custom currency
   */
  @post
  @operationId("create-cost-basis")
  @summary("Create cost basis")
  create_cost_basis(
    @header contentType: "application/json",
    @body body: CostBasis,
    @path currencyId: Shared.ULID,
  ): Shared.CreateResponse<CostBasis> | Common.ErrorResponses;
}
