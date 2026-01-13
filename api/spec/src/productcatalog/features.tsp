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
