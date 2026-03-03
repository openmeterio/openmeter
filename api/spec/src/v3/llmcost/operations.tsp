import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../common/error.tsp";
import "../common/parameters.tsp";
import "../shared/index.tsp";
import "./prices.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace LLMCost;

/**
 * Filter options for listing LLM cost prices.
 */
@friendlyName("ListLLMCostPricesParamsFilter")
model ListPricesParamsFilter {
  /** Filter by provider (case-insensitive, partial match). */
  provider?: string;

  /** Filter by model ID (case-insensitive, partial match). */
  model_id?: string;

  /** Filter by model name (case-insensitive, partial match). */
  model_name?: string;

  /** Filter by currency code (case-insensitive, partial match). */
  currency?: Shared.CurrencyCode;
}

/**
 * Operations for managing global LLM cost prices.
 */
interface LLMCostPricesOperations {
  /**
   * List global LLM cost prices.
   * Returns prices with overrides applied if any.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-llm-cost-prices")
  @summary("List LLM cost prices")
  list_prices(
    /**
     * Filter prices.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListPricesParamsFilter,

    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Price> | Common.ErrorResponses;

  /**
   * Get a specific LLM cost price by ID.
   * Returns the price with overrides applied if any.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-llm-cost-price")
  @summary("Get LLM cost price")
  get_price(
    @path priceId: Shared.ULID,
  ): Price | Common.NotFound | Common.ErrorResponses;
}

/**
 * Operations for managing per-namespace price overrides.
 */
interface LLMCostOverridesOperations {
  /**
   * List per-namespace price overrides.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("list-llm-cost-overrides")
  @summary("List LLM cost overrides")
  list_overrides(
    @query(#{ style: "deepObject", explode: true })
    filter?: ListPricesParamsFilter,

    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Price> | Common.ErrorResponses;

  /**
   * Create a per-namespace price override.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-llm-cost-override")
  @summary("Create LLM cost override")
  create_override(
    @body body: OverrideCreate,
  ): Shared.CreateResponse<Price> | Common.ErrorResponses;

  /**
   * Delete a per-namespace price override.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @delete
  @operationId("delete-llm-cost-override")
  @summary("Delete LLM cost override")
  delete_override(
    @path priceId: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}
