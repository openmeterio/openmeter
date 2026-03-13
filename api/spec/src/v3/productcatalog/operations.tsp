import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./ratecard.tsp";
import "./plan.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace ProductCatalog;

interface ProductCatalogOperations {
  /**
   * List prices.
   */
  @get
  @route("/prices")
  @operationId("list-prices")
  @summary("List prices")
  list(): RateCard | Common.ErrorResponses;

  /**
   * Test List plans.
   */
  @get
  @route("/plans")
  @operationId("list-plans")
  @summary("List plans")
  listPlans(): Plan | Common.ErrorResponses;
}
