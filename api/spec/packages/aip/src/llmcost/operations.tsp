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
 * A filter for a single string field.
 * TODO: This is a temporary solution to support the filter API.
 */
@friendlyName("FilterSingleString")
// @useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/StringFieldFilter")
model FilterSingleString {
  /**
   * The field must match the provided value.
   */
  eq?: string;

  /**
   * The field must match the provided value. This is an "override" filter that can be used when the default `eq` filter is not sufficient, e.g. when filtering by `model_name` where the name can have multiple variants for the same model ID.
   */
  oeq?: string;

  /**
   * The field must not match the provided value.
   */
  neq?: string;

  /**
   * The field must contain the provided value.
   */
  contains?: string;

  /**
   * The field must contain the provided value. This is an "override" filter that can be used when the default `contains` filter is not sufficient.
   */
  ocontains?: string;
}

/**
 * Filter options for listing LLM cost prices.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "model_id and model_name should not be grouped"
@friendlyName("ListLLMCostPricesParamsFilter")
model ListPricesParamsFilter {
  /** Filter by provider. e.g. ?filter[provider][eq]=openai */
  @extension("x-go-type", "FilterString")
  provider?: string | FilterSingleString;

  /** Filter by model ID. e.g. ?filter[model_id][eq]=gpt-4 */
  @extension("x-go-type", "FilterString")
  model_id?: string | FilterSingleString;

  /** Filter by model name. e.g. ?filter[model_name][contains]=gpt */
  @extension("x-go-type", "FilterString")
  model_name?: string | FilterSingleString;

  /** Filter by currency code. e.g. ?filter[currency][eq]=USD */
  @extension("x-go-type", "FilterString")
  currency?: string | FilterSingleString;
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
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-llm-cost-prices")
  @summary("List LLM cost prices")
  list_prices(
    /**
     * Filter prices.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListPricesParamsFilter,

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
    @query(#{ style: "deepObject", explode: true })
    filter?: ListPricesParamsFilter,

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
