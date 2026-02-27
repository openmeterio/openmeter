import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Order by options for features.
 */
@friendlyName("FeatureOrderBy")
enum FeatureOrderBy {
  ID: "id",
  Key: "key",
  Name: "name",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  CreatedAt: "createdAt",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  UpdatedAt: "updatedAt",
}

/**
 * Represents a feature that can be enabled or disabled for a plan.
 * Used both for product catalog and entitlements.
 */
@friendlyName("FeatureCreateInputs")
model FeatureCreateInputs {
  @summary("The unique key of the feature")
  key: Key;

  @summary("The human-readable name of the feature")
  name: string;

  @summary("Optional metadata")
  @example(#{ key: "value" })
  metadata?: Metadata;

  // /**
  //  * The meter that the feature is associated with and and based on which usage is calculated.
  //  * The meter selected must have one of the following aggregations: SUM, COUNT, UNIQUE_COUNT, LATEST
  //  */
  // @summary("Meter key")
  // @example("tokens_total")
  // meterKey?: Key;

  @summary("Meter slug")
  @example("tokens_total")
  meterSlug?: Key;

  /**
   * Optional meter group by filters.
   * Useful if the meter scope is broader than what feature tracks.
   * Example scenario would be a meter tracking all token use with groupBy fields for the model,
   * then the feature could filter for model=gpt-4.
   *
   * ⚠️ __Deprecated__: Use advancedMeterGroupByFilters instead
   */
  #deprecated "Use advancedMeterGroupByFilters instead"
  @summary("Meter group by filters")
  @example(#{ `model`: "gpt-4", type: "input" })
  meterGroupByFilters?: Record<string>;

  /**
   * Optional advanced meter group by filters.
   * You can use this to filter for values of the meter groupBy fields.
   */
  @summary("Advanced meter group by filters")
  @example(#{
    `model`: #{ $in: #["gpt-4", "gpt-4o"] },
    type: #{ $eq: "input" },
  })
  advancedMeterGroupByFilters?: Record<FilterString>;

  /**
   * Optional per-unit cost configuration.
   * Use "manual" for a fixed per-unit cost, or "llm" to look up cost
   * from the LLM cost database based on meter group-by properties.
   */
  @summary("Unit cost")
  unitCost?: FeatureUnitCost;
}

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
  amount: Numeric;
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
  providerProperty?: string;

  /**
   * Static LLM provider value (e.g., "openai", "anthropic").
   * Use this when the feature tracks a single provider.
   * Mutually exclusive with `providerProperty`.
   */
  @summary("Provider")
  provider?: string;

  /**
   * Meter group-by property that holds the model ID.
   * Use this when the meter has a group-by dimension for model.
   * Mutually exclusive with `model`.
   */
  @summary("Model property")
  modelProperty?: string;

  /**
   * Static model ID value (e.g., "gpt-4", "claude-3-5-sonnet").
   * Use this when the feature tracks a single model.
   * Mutually exclusive with `modelProperty`.
   */
  @summary("Model")
  `model`?: string;

  /**
   * Meter group-by property that holds the token type.
   * Use this when the meter has a group-by dimension for token type.
   * Mutually exclusive with `tokenType`.
   */
  @summary("Token type property")
  tokenTypeProperty?: string;

  /**
   * Static token type value.
   * Use this when the feature tracks a single token type (e.g., only input tokens).
   * Expected values: input, output, input_cached, reasoning, cache_write.
   * Mutually exclusive with `tokenTypeProperty`.
   */
  @summary("Token type")
  tokenType?: string;

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
  inputPerToken: Numeric;

  /** Cost per output token in USD. */
  @summary("Output per token")
  outputPerToken: Numeric;

  /** Cost per cached input token in USD. */
  @summary("Input cached per token")
  inputCachedPerToken?: Numeric;

  /** Cost per reasoning token in USD. */
  @summary("Reasoning per token")
  reasoningPerToken?: Numeric;

  /** Cost per cache write token in USD. */
  @summary("Cache write per token")
  cacheWritePerToken?: Numeric;
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

/**
 * A row in the result of a feature cost query.
 */
@friendlyName("FeatureCostQueryRow")
model FeatureCostQueryRow {
  /** The metered usage value for the period. */
  usage: Numeric;

  /** The unit cost used for cost computation. Null when pricing is not available for the group-by combination. */
  unitCost: Numeric | null;

  /** The computed cost amount (usage × unit cost). Null when pricing is not available for the group-by combination. */
  cost: Numeric | null;

  /** The currency code of the cost amount. */
  currency: CurrencyCode;

  /** Detail message when cost amount is null, explaining why the cost could not be resolved. */
  detail?: string;

  /** The start of the window the value is aggregated over. */
  windowStart: DateTime;

  /** The end of the window the value is aggregated over. */
  windowEnd: DateTime;

  /** The subject the value is aggregated over. */
  subject?: string;

  /** The customer ID the value is aggregated over. */
  customerId?: string;

  /** The group by values the value is aggregated over. */
  groupBy?: Record<string | null>;
}

/**
 * Result of a feature cost query.
 */
@friendlyName("FeatureCostQueryResult")
model FeatureCostQueryResult {
  /** Start of the queried period. */
  from?: DateTime;

  /** End of the queried period. */
  to?: DateTime;

  /** The window size that the cost is aggregated. */
  windowSize?: WindowSize;

  /** The currency code of the cost amounts. */
  currency: CurrencyCode;

  /** The cost data rows. */
  data: FeatureCostQueryRow[];
}

/**
 * List features result
 */
@oneOf
@friendlyName("ListFeaturesResult")
union ListFeaturesResult {
  Feature[],
  PaginatedResponse<Feature>,
}

/**
 * Represents a feature that can be enabled or disabled for a plan.
 * Used both for product catalog and entitlements.
 */
@friendlyName("Feature")
model Feature {
  ...ResourceTimestamps;
  ...Archiveable;
  ...FeatureCreateInputs;

  /**
   * Readonly unique ULID identifier.
   */
  @example("01ARZ3NDEKTSV4RRFFQ69G5FAV")
  @visibility(Lifecycle.Read)
  id: ULID;
}
