import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "../currency.tsp";
import "./cost-bases.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Currencies;


interface CostBasesOperations {
  /**
   * Cost Bases operations
   */

  /**
   * Get cost bases by currency id
   */
  @get
  @operationId("get-cost-bases-by-id")
  @summary("Get cost bases by id")
  get_cost_bases(
    @path currencyId: Shared.ULID
  ): CostBases | Common.ErrorResponses
}