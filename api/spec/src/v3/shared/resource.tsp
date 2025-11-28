import "../common/properties.tsp";
import "./properties.tsp";

namespace Shared;

/**
 * Represents common fields of resources.
 */
@friendlyName("Resource")
model Resource {
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  id: ULID;

  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  @maxLength(1024)
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  description?: string;

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  labels?: Common.Labels;

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  public_labels?: Common.Labels;

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  created_at?: Common.CreatedAt;

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  updated_at?: Common.UpdatedAt;

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  deleted_at?: Common.DeletedAt;
}

/**
 * Represents common fields of resources with a key.
 */
@friendlyName("ResourceWithKey")
model ResourceWithKey {
  ...Resource;

  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read, Lifecycle.Create)
  key: ResourceKey;
}
