import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.OpenAPI;

namespace Common;

/**
 * Labels store metadata of an entity that can be used for filtering an entity list or for searching across entity types.
 *
 * Keys must be of length 1-63 characters, and cannot start with "kong", "konnect", "mesh", "kic", or "_".
 */
@useRef("../../../../common/definitions/konnect_properties.yaml#/components/schemas/Labels")
@friendlyName("Labels")
model Labels {}

/**
 * Public labels store information about an entity that can be used for filtering a list of objects.
 */
@useRef("../../../../common/definitions/konnect_properties.yaml#/components/schemas/PublicLabels")
@friendlyName("PublicLabels")
model PublicLabels {}
