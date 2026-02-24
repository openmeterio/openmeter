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
   * Filter currencies by type.
   */
  type?: CurrencyType;
}

interface CurrenciesOperations {
  /**
   * List currencies supported by the billing system.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-currencies")
  @summary("List currencies")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Filter currencies returned in the response.
     *
     * To filter currencies by type add the following query param: filter[type]=custom
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListCurrenciesParamsFilter,
  ): Shared.PagePaginatedResponse<Currency> | Common.ErrorResponses;
}

interface CurrenciesCustomOperations {
  /**
   * Create a custom currency.
   * This operation allows defining your own custom currency for billing purposes.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-custom-currency")
  @summary("Create custom currency")
  create(
    @body body: Shared.CreateRequest<CurrencyCustom>,
  ): Shared.CreateResponse<CurrencyCustom> | Common.ErrorResponses;
}
