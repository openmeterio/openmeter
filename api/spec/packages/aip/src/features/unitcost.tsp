import "../shared/index.tsp";

namespace Features;

/**
 * The type of unit cost.
 */
@friendlyName("FeatureUnitCostType")
enum FeatureUnitCostType {
  llm: "llm",
  manual: "manual",
}

/**
 * A fixed per-unit cost amount.
 */
@friendlyName("FeatureManualUnitCost")
model FeatureManualUnitCost {
  type: FeatureUnitCostType.manual;

  /** Fixed per-unit cost amount in USD. */
  amount: Shared.Numeric;
}

/**
 * LLM cost lookup configuration.
 * Maps meter group-by dimensions to LLM cost database fields.
 */
@friendlyName("FeatureLLMUnitCost")
model FeatureLLMUnitCost {
  type: FeatureUnitCostType.llm;

  /**
   * Meter group-by property that holds the LLM provider.
   * Use this when the meter has a group-by dimension for provider.
   * Mutually exclusive with `provider`.
   */
  @summary("Provider property")
  provider_property?: string;

  /**
   * Static LLM provider value (e.g., "openai", "anthropic").
   * Use this when the feature tracks a single provider.
   * Mutually exclusive with `provider_property`.
   */
  @summary("Provider")
  provider?: string;

  /**
   * Meter group-by property that holds the model ID.
   * Use this when the meter has a group-by dimension for model.
   * Mutually exclusive with `model`.
   */
  @summary("Model property")
  model_property?: string;

  /**
   * Static model ID value (e.g., "gpt-4", "claude-3-5-sonnet").
   * Use this when the feature tracks a single model.
   * Mutually exclusive with `model_property`.
   */
  @summary("Model")
  `model`?: string;

  /**
   * Meter group-by property that holds the token type.
   * Use this when the meter has a group-by dimension for token type.
   * Mutually exclusive with `token_type`.
   */
  @summary("Token type property")
  token_type_property?: string;

  /**
   * Static token type value.
   * Use this when the feature tracks a single token type (e.g., only input tokens).
   * Expected values: input, output, cache_read, reasoning, cache_write, request, response.
   * `request` is an alias for `input`, `response` is an alias for `output`.
   * Mutually exclusive with `token_type_property`.
   */
  @summary("Token type")
  token_type?: string;

  /**
   * Resolved per-token pricing from the LLM cost database.
   * Only populated in responses when the feature's meter group-by filters
   * specify exact provider and model values.
   */
  @summary("Resolved pricing")
  @visibility(Lifecycle.Read)
  pricing?: FeatureLLMUnitCostPricing;
}

/**
 * Resolved per-token pricing from the LLM cost database.
 */
@friendlyName("FeatureLLMUnitCostPricing")
model FeatureLLMUnitCostPricing {
  /** Cost per input token in USD. */
  @summary("Input per token")
  input_per_token: Shared.Numeric;

  /** Cost per output token in USD. */
  @summary("Output per token")
  output_per_token: Shared.Numeric;

  /** Cost per cache read token in USD. */
  @summary("Cache read per token")
  cache_read_per_token?: Shared.Numeric;

  /** Cost per reasoning token in USD. */
  @summary("Reasoning per token")
  reasoning_per_token?: Shared.Numeric;

  /** Cost per cache write token in USD. */
  @summary("Cache write per token")
  cache_write_per_token?: Shared.Numeric;
}

/**
 * Per-unit cost configuration for a feature.
 * Either a fixed manual amount or a dynamic LLM cost lookup.
 */
@friendlyName("FeatureUnitCost")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union FeatureUnitCost {
  @summary("Manual unit cost")
  manual: FeatureManualUnitCost,

  @summary("LLM unit cost")
  llm: FeatureLLMUnitCost,
}
