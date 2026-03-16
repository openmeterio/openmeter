import "../shared/index.tsp";
import "../meters/index.tsp";
import "@typespec/json-schema";

using TypeSpec.JsonSchema;

namespace ProductCatalog;

/**
 * A capability or billable dimension offered by a provider.
 */
@friendlyName("Feature")
model Feature {
  ...Shared.ResourceWithKey;

  /**
   * The meter that the feature is associated with and and based on which usage is calculated.
   * If not specified, the feature is static.
   */
  @summary("Meter reference")
  @visibility(Lifecycle.Read, Lifecycle.Create)
  meter?: {
    ...Shared.ResourceWithKeyReference<Meters.Meter>;

    /**
     * Filters to apply to the dimensions of the meter.
     */
    @maxProperties(10)
    @summary("Meter dimensions filters")
    filters?: Record<Shared.QueryFilterStringMapItem>;
  };
}
