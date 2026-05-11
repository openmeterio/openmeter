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
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  type?: CurrencyType;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "filter field"
  code?: Common.StringFieldFilterExact;
}

interface CurrenciesOperations {
  /**
   * List currencies supported by the billing system.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-currencies")
  @summary("List currencies")
  list(
    ...Common.PagePaginationQuery,

    /**
     * Sort currencies returned in the response. Supported sort attributes are:
     *
     * - `code` (default)
     * - `name`
     *
     * The `asc` suffix is optional as the default sort order is ascending. The `desc`
     * suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

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
   * Create a custom currency. This operation allows defining your own custom
   * currency for billing purposes.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @operationId("create-custom-currency")
  @summary("Create custom currency")
  create(
    @body body: Shared.CreateRequest<CurrencyCustom>,
  ): Shared.CreateResponse<CurrencyCustom> | Common.ErrorResponses;
}
