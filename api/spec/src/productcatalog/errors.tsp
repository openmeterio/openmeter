import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";

namespace OpenMeter.ProductCatalog;

/**
 * ValidationResource captures the resource information attached to ValidationError.
 */
@friendlyName("ValidationResource")
model ValidationResource {
  /**
   * The key of the resource.
   */
  @visibility(Lifecycle.Read)
  @example("token")
  key: string;

  /**
   * The resource type.
   */
  @visibility(Lifecycle.Read)
  @example("ratecard")
  kind: string;

  /**
   * Additional attributes.
   */
  @visibility(Lifecycle.Read)
  attributes?: Annotations;
}

/**
 * Validation errors providing details about compatibility issues between a plan and its add-on.
 */
@friendlyName("ValidationError")
model ValidationError {
  /**
   * The resource information.
   */
  @visibility(Lifecycle.Read)
  resource: ValidationResource;

  /**
   * The field of the resource the error relates to.
   */
  @visibility(Lifecycle.Read)
  @example("key")
  field: string;

  /**
   * The detailed description of the error.
   */
  @visibility(Lifecycle.Read)
  @example("invalid key")
  detail: string;
}
