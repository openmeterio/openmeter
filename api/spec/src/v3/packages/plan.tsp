import "@typespec/openapi";
import "@typespec/openapi3";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace Packages;

/**
 * Reference to a plan by its id or key and version.
 */
@oneOf
@friendlyName("PlanReference")
union Reference {
  id: {
    id: Shared.ULID,
  },
  keyVersion: {
    key: Shared.ResourceKey,
    version?: integer,
  },
}
