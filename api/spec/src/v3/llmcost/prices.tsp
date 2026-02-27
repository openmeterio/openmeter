import "../shared/index.tsp";

namespace LLMCost;

/**
 * Identifies where an LLM cost price came from.
 */
@friendlyName("LLMCostPriceSource")
enum PriceSource {
  /** Price was manually configured by a user. */
  manual,

  /** Price was automatically synced from external pricing sources. */
  system,
}

/**
 * Token pricing for an LLM model, denominated in USD per token.
 */
@friendlyName("LLMCostModelPricing")
model ModelPricing {
  /** Input price per token (USD). */
  input_per_token: Shared.Numeric;

  /** Output price per token (USD). */
  output_per_token: Shared.Numeric;

  /** Cached input price per token (USD). */
  input_cached_per_token?: Shared.Numeric;

  /** Reasoning output price per token (USD). */
  reasoning_per_token?: Shared.Numeric;

  /** Cache write price per token (USD). */
  cache_write_per_token?: Shared.Numeric;
}

/**
 * An LLM cost price record, representing the cost per token
 * for a specific model from a specific provider.
 */
@friendlyName("LLMCostPrice")
model Price {
  /** Unique identifier. */
  @visibility(Lifecycle.Read)
  id: Shared.ULID;

  /** Provider/vendor of the model (e.g., "openai", "anthropic"). */
  @visibility(Lifecycle.Read)
  provider: string;

  /** Canonical model identifier (e.g., "gpt-4", "claude-3-5-sonnet"). */
  @visibility(Lifecycle.Read)
  model_id: string;

  /** Human-readable model name. */
  @visibility(Lifecycle.Read)
  model_name: string;

  /** Token pricing data. */
  @visibility(Lifecycle.Read)
  pricing: ModelPricing;

  /** Currency code (currently always "USD"). */
  @visibility(Lifecycle.Read)
  currency: string;

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
  created_at?: Shared.DateTime;

  /** Last update timestamp. */
  @visibility(Lifecycle.Read)
  updated_at?: Shared.DateTime;
}

/**
 * Input for creating a per-namespace price override.
 */
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

  /** When this override becomes effective. */
  effective_from: Shared.DateTime;

  /** When this override expires. */
  effective_to?: Shared.DateTime;
}

/**
 * Input for updating a per-namespace price override.
 */
@friendlyName("LLMCostOverrideUpdate")
model OverrideUpdate {
  /** Updated token pricing data. */
  pricing?: ModelPricing;

  /** Updated expiry time. */
  effective_to?: Shared.DateTime;
}
