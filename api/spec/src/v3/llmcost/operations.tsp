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
  /** Filter by provider. */
  provider?: string;

  /** Filter by model ID. */
  model_id?: string;
}

/**
 * Operations for managing global LLM cost prices.
 */
interface LLMCostPricesOperations {
  /**
   * List global LLM cost prices.
   * Returns the global (synced) price database of LLM model costs.
   */
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

    /** Point-in-time query: return prices effective at this time. */
    @query at?: Shared.DateTime,

    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Price> | Common.ErrorResponses;

  /**
   * Get a specific LLM cost price by ID.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @operationId("get-llm-cost-price")
  @summary("Get LLM cost price")
  get_price(
    @path priceId: Shared.ULID,
  ): Price | Common.NotFound | Common.ErrorResponses;

  /**
   * Resolve the effective price for a model in the current namespace.
   * Returns namespace override if available, otherwise the global price.
   */
  @extension(Shared.UnstableExtension, true)
  @get
  @route("resolve")
  @operationId("resolve-llm-cost-price")
  @summary("Resolve LLM cost price")
  resolve_price(
    /** Provider to resolve price for. */
    @query provider: string,

    /** Model ID to resolve price for. */
    @query model_id: string,

    /** Point-in-time: resolve price effective at this time. */
    @query at?: Shared.DateTime,
  ): Price | Common.NotFound | Common.ErrorResponses;
}

/**
 * Operations for managing per-namespace price overrides.
 */
interface LLMCostOverridesOperations {
  /**
   * List per-namespace price overrides.
   */
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
  @extension(Shared.UnstableExtension, true)
  @post
  @operationId("create-llm-cost-override")
  @summary("Create LLM cost override")
  create_override(
    @body body: OverrideCreate,
  ): Shared.CreateResponse<Price> | Common.ErrorResponses;

  /**
   * Update a per-namespace price override.
   */
  @extension(Shared.UnstableExtension, true)
  @put
  @operationId("update-llm-cost-override")
  @summary("Update LLM cost override")
  update_override(
    @path overrideId: Shared.ULID,
    @body body: OverrideUpdate,
  ): Price | Common.NotFound | Common.ErrorResponses;

  /**
   * Delete a per-namespace price override.
   */
  @extension(Shared.UnstableExtension, true)
  @delete
  @operationId("delete-llm-cost-override")
  @summary("Delete LLM cost override")
  delete_override(
    @path overrideId: Shared.ULID,
  ): void | Common.NotFound | Common.ErrorResponses;
}
