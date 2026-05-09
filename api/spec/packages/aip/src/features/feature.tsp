import "../shared/index.tsp";
import "../meters/index.tsp";
import "./unitcost.tsp";
import "@typespec/json-schema";

using TypeSpec.JsonSchema;

namespace Features;

@friendlyName("FeatureMeterReference")
model FeatureMeterReference {
  /**
   * The ID of the meter to associate with this feature.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Meter ID")
  id: Shared.ULID;

  /**
   * Filters to apply to the dimensions of the meter.
   */
  @maxProperties(10)
  @summary("Meter dimensions filters")
  filters?: Record<Shared.QueryFilterStringMapItem>;
}

/**
 * A capability or billable dimension offered by a provider.
 */
@friendlyName("Feature")
model Feature {
  ...Shared.ResourceWithKey;

  /**
   * The meter that the feature is associated with and and based on which usage is
   * calculated. If not specified, the feature is static.
   */
  @summary("Meter reference")
  @visibility(Lifecycle.Read, Lifecycle.Create)
  meter?: FeatureMeterReference;

  /**
   * Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
   * "llm" to look up cost from the LLM cost database based on meter group-by
   * properties.
   */
  @summary("Unit cost")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  unit_cost?: FeatureUnitCost;
}

/**
 * Request body for updating a feature. Currently only the unit_cost field can be
 * updated.
 */
@friendlyName("UpdateFeatureRequest")
model FeatureUpdateRequest {
  /**
   * Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
   * "llm" to look up cost from the LLM cost database based on meter group-by
   * properties.
   */
  @summary("Unit cost")
  unit_cost?: FeatureUnitCost | null;
}
