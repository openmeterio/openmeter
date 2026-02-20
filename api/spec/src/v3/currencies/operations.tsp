import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./currency.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * List currencies response.
 */
@friendlyName("CurrenciesResponse")
model CurrenciesResponse {
  @Http.statusCode _: 200;

  @pageItems
  data: Currency[];
}

@extension(Shared.PrivateExtension, true)
@extension(Shared.UnstableExtension, true)
interface CurrenciesOperations {
  /**
   * List all supported currencies.
   */
  @get
  @operationId("list-currencies")
  @summary("List currencies")
  list(): CurrenciesResponse | Common.ErrorResponses;

  @post
  @operationId("create-custom-currency")
  @summary("Create custom currency")
  create(
    @header contentType: "application/json",
    @body body: Currencies.CustomCurrency,
  ): Shared.CreateResponse<CustomCurrency> | Common.ErrorResponses;
}
