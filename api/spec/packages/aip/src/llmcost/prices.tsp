import "../shared/index.tsp";

namespace LLMCost;

/**
 * Identifies where an LLM cost price came from.
 */
@friendlyName("LLMCostPriceSource")
enum PriceSource {
  /** Price was manually configured by a user. */
  Manual: "manual",

  /** Price was automatically synced from external pricing sources. */
  System: "system",
}

/**
 * Token pricing for an LLM model, denominated per token.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "per token prices should be grouped together"
@friendlyName("LLMCostModelPricing")
model ModelPricing {
  /** Input price per token (USD). */
  input_per_token: Shared.Numeric;

  /** Output price per token (USD). */
  output_per_token: Shared.Numeric;

  /** Cache read price per token (USD). */
  cache_read_per_token?: Shared.Numeric;

  /** Cache write price per token (USD). */
  cache_write_per_token?: Shared.Numeric;

  /** Reasoning output price per token (USD). */
  reasoning_per_token?: Shared.Numeric;
}

/**
 * An LLM cost price record, representing the cost per token
 * for a specific model from a specific provider.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "source and effective_from/to should be grouped together"
@friendlyName("LLMCostPrice")
model Price {
  /** Unique identifier. */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /** Provider of the model. */
  @visibility(Lifecycle.Read)
  provider: Provider;

  /** The model. */
  @visibility(Lifecycle.Read)
  `model`: Model;

  /** Token pricing data. */
  @visibility(Lifecycle.Read)
  pricing: ModelPricing;

  /** Currency code (currently always "USD"). */
  @visibility(Lifecycle.Read)
  currency: Shared.CurrencyCode;

  /** Where this price came from. */
  @visibility(Lifecycle.Read)
  source: PriceSource;

  /** When this price becomes effective. */
  @visibility(Lifecycle.Read)
  effective_from: Shared.DateTime;

  /** When this price expires (null = current). */
  @visibility(Lifecycle.Read)
  effective_to?: Shared.DateTime;

  /** Creation timestamp. */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /** Last update timestamp. */
  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;
}

/**
 * LLM Provider
 */
@friendlyName("LLMCostProvider")
model Provider {
  /** Identifier of the provider, e.g., "openai", "anthropic". */
  id: string;

  /** Name of the provider, e.g., "OpenAI", "Anthropic". */
  name: string;
}

/**
 * LLM Model
 */
@friendlyName("LLMCostModel")
model Model {
  /** Identifier of the model, e.g., "gpt-4", "claude-3-5-sonnet". */
  id: string;

  /** Name of the model, e.g., "GPT-4", "Claude 3.5 Sonnet". */
  name: string;
}

/**
 * Input for creating a per-namespace price override.
 * Unique per provider, model and currency.
 * If an override already exists for the given provider, model and currency, it will be updated.
 * If an override does not exist, it will be created.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "model_id, model_name, and effective_from/to should be grouped together"
@friendlyName("LLMCostOverrideCreate")
model OverrideCreate {
  /** Provider/vendor of the model. */
  provider: string;

  /** Canonical model identifier. */
  model_id: string;

  /** Human-readable model name. */
  model_name?: string;

  /** Token pricing data. */
  pricing: ModelPricing;

  /** Currency code. */
  currency: Shared.CurrencyCode;

  /** When this override becomes effective. */
  effective_from: Shared.DateTime;

  /** When this override expires. */
  effective_to?: Shared.DateTime;
}
