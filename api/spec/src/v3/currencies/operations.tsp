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
 * Filter options for listing currencies.
 */
@friendlyName("ListCurrenciesParamsFilter")
model ListCurrenciesParamsFilter {
  /**
   * Filter currencies by code.
   */
}

/**
 * List currencies response.
 */
@friendlyName("CurrenciesResponse")
model CurrenciesResponse {
  @Http.statusCode _: 200;

  @pageItems
  data: Currency[];
}

interface CurrenciesOperations {
  /**
   * List all supported currencies.
   */
  @get
  @operationId("list-currencies")
  @summary("List currencies")
  list(
    // /**
    //  * Filter currencies returned in the response.
    //  */
    // @query(#{ style: "deepObject", explode: true })
    // filter?: ListCurrenciesParamsFilter,
  ): CurrenciesResponse | Common.ErrorResponses;
}
