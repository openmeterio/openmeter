import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./ratecard.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace ProductCatalog;

interface ProductCatalogOperations {
  /**
   * List prices.
   */
  @get
  @operationId("list-prices")
  @summary("List prices")
  list(): RateCard | Common.ErrorResponses;
}
