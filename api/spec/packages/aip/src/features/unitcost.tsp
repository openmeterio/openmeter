import "../shared/index.tsp";

namespace Features;

/**
 * The type of unit cost.
 */
@friendlyName("BillingFeatureUnitCostType")
enum FeatureUnitCostType {
  llm: "llm",
  manual: "manual",
}

/**
 * A fixed per-unit cost amount.
 */
@friendlyName("BillingFeatureManualUnitCost")
model FeatureManualUnitCost {
  type: FeatureUnitCostType.manual;

  /** Fixed per-unit cost amount in USD. */
  amount: Shared.Numeric;
}

/**
 * Token type for LLM cost lookup.
 */
@friendlyName("BillingFeatureLLMTokenType")
enum FeatureLLMTokenType {
  input: "input",
  output: "output",
  cache_read: "cache_read",
  cache_write: "cache_write",
  reasoning: "reasoning",

  /** Alias for `input`. */
  request: "request",

  /** Alias for `output`. */
  response: "response",
}

/**
 * LLM cost lookup configuration.
 * Each dimension (provider, model, token type) can be specified as either
 * a static value or a meter group-by property name (mutually exclusive).
 */
@friendlyName("BillingFeatureLLMUnitCost")
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
   * `request` is an alias for `input`, `response` is an alias for `output`.
   * Mutually exclusive with `token_type_property`.
   */
  @summary("Token type")
  token_type?: FeatureLLMTokenType;

  /**
   * Resolved per-token pricing from the LLM cost database.
   * Populated in responses when the provider and model can be determined,
   * either from static values or from meter group-by filters with exact matches.
   */
  @summary("Resolved pricing")
  @visibility(Lifecycle.Read)
  pricing?: FeatureLLMUnitCostPricing;
}

/**
 * Resolved per-token pricing from the LLM cost database.
 */
@friendlyName("BillingFeatureLLMUnitCostPricing")
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
@friendlyName("BillingFeatureUnitCost")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union FeatureUnitCost {
  @summary("Manual unit cost")
  manual: FeatureManualUnitCost,

  @summary("LLM unit cost")
  llm: FeatureLLMUnitCost,
}
