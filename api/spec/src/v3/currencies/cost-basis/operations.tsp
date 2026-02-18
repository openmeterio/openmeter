import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "../currency.tsp";
import "./cost_basis.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * Create cost basis response
 */
@friendlyName("CreateCostBasisResponse")
model CreateCostBasisResponse {
  @statusCode _: 201;
}

interface CostBasisOperations {
  /**
   * Cost Basis operations.
   */
  @post
  @operationId("create-cost-basis")
  @summary("Create custom cost basis for custom currencies")
  create_cost_basis(
    @header contentType: "application/json",
    @body body: CostBasis
  ): CreateCostBasisResponse | Common.ErrorResponses;
}