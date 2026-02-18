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
   * Get cost bases
   */
  @get
  @operationId("list-cost-bases")
  @summary("List cost bases")
  list_cost_bases(): CostBases | Common.ErrorResponses;
}