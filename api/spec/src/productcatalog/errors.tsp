import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

/**
 * Validation errors providing details about compatibility issues between a plan and its add-on.
 */
@friendlyName("ValidationError")
model ValidationError {
  /**
   * The path to the field.
   */
  @visibility(Lifecycle.Read)
  @example("addons/pro/ratecards/token/featureKey")
  field: string;

  /**
   * The machine readable description of the error.
   */
  @visibility(Lifecycle.Read)
  @example("invalid_feature_key")
  code: string;

  /**
   * The human readable description of the error.
   */
  @visibility(Lifecycle.Read)
  @example("not found feature by key")
  message: string;

  /**
   * Additional attributes.
   */
  @visibility(Lifecycle.Read)
  attributes?: Annotations;
}
