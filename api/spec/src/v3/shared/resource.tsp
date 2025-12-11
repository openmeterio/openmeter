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
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  labels?: Common.Labels;

  // TODO: is this needed?
  #suppress "@openmeter/api-spec/casing" ""
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  public_labels?: Common.PublicLabels;

  #suppress "@openmeter/api-spec/casing" ""
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  @visibility(Lifecycle.Read)
  created_at?: DateTime;

  #suppress "@openmeter/api-spec/casing" ""
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  @visibility(Lifecycle.Read)
  updated_at?: DateTime;

  #suppress "@openmeter/api-spec/casing" ""
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  @visibility(Lifecycle.Read)
  deleted_at?: DateTime;
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
