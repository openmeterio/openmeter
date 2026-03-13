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
 * AIP-160 filter expression in deep-object URL format.
 * e.g. ?filter[provider][eq]=openai
 */
@extension("x-go-type", "aip160.Filter")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/pkg/filter/aip160" }
)
scalar AIP160Filter extends string;

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
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-llm-cost-prices")
  @summary("List LLM cost prices")
  list_prices(
    /**
     * AIP-160 filter expression using the deep-object format.
     *
     * Supported fields: `provider`, `model_id`, `model_name`, `currency`
     *
     * Supported operators: `eq`, `neq`, `contains`, `oeq`, `ocontains`
     *
     * Examples:
     * - `?filter[provider][eq]=openai`
     * - `?filter[model_id][eq]=gpt-4`
     * - `?filter[model_name][contains]=gpt`
     * - `?filter[currency][eq]=USD`
     */
    @query filter?: AIP160Filter,

    /**
     * Sort prices returned in the response.
     * Supported sort attributes are:
     * - `id`
     * - `provider.id`
     * - `model.id` (default)
     * - `effective_from`
     * - `effective_to`
     *
     * The `asc` suffix is optional as the default sort order is ascending.
     * The `desc` suffix is used to specify a descending order.
     */
    @query(#{ name: "sort" })
    sort?: Common.SortQuery,

    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Price> | Common.ErrorResponses;

  /**
   * Get a specific LLM cost price by ID.
   * Returns the price with overrides applied if any.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
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
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-llm-cost-overrides")
  @summary("List LLM cost overrides")
  list_overrides(
    /**
     * AIP-160 filter expression using the deep-object format.
     *
     * Supported fields: `provider`, `model_id`, `model_name`, `currency`
     *
     * Supported operators: `eq`, `neq`, `contains`, `oeq`, `ocontains`
     *
     * Examples:
     * - `?filter[provider][eq]=openai`
     * - `?filter[model_id][eq]=gpt-4`
     * - `?filter[model_name][contains]=gpt`
     * - `?filter[currency][eq]=USD`
     */
    @query filter?: AIP160Filter,

    ...Common.PagePaginationQuery,
  ): Shared.PagePaginatedResponse<Price> | Common.ErrorResponses;

  /**
   * Create a per-namespace price override.
   */
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
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
  @extension(Shared.InternalExtension, true)
  @delete
  @operationId("delete-llm-cost-override")
  @summary("Delete LLM cost override")
  delete_override(
    @path priceId: Shared.ULID,
  ): Shared.DeleteResponse | Common.NotFound | Common.ErrorResponses;
}
