import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "../currency.tsp";
import "./cost-basis.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Currencies;

/**
 * Filter options for listing cost bases.
 */
@friendlyName("ListCostBasesParamsFilter")
model ListCostBasesParamsFilter {
  /**
   * Filter cost bases by fiat currency code.
   */
  fiat_code?: Shared.CurrencyCode;
}

interface CurrenciesCustomCostBasesOperations {
  /**
   * List cost bases for a currency.
   * For custom currencies, there can be multiple cost bases with different `effective_from` dates.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-cost-bases")
  @summary("List cost bases")
  get_cost_bases(
    @path currencyId: Shared.ULID,

    /**
     * Filter cost bases returned in the response.
     *
     * To filter cost bases by fiat currency code add the following query param: filter[fiat_code]=USD
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListCostBasesParamsFilter,

    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<CostBasis> | Common.NotFound | Common.ErrorResponses;

  /**
   * Create a cost basis for a currency.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-cost-basis")
  @summary("Create cost basis")
  create_cost_basis(
    @path currencyId: Shared.ULID,
    @body body: Shared.CreateRequest<CostBasis>,
  ): Shared.CreateResponse<CostBasis> | Common.ErrorResponses;
}
