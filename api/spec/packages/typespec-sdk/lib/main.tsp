import "./index.js";

using TypeSpec.Reflection;

namespace OpenMeter.Sdk;

/**
 * Selects the SDK query codec and its encoded transport type for a query
 * parameter. This metadata is consumed by OpenMeter's language emitters and is
 * intentionally invisible to the OpenAPI emitter.
 */
extern dec queryCodec(
  target: ModelProperty,
  codec: valueof "sort",
  wireType: Scalar
);
