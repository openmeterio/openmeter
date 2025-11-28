import "@typespec/rest";

namespace TypeSpec.Rest.Resource {
  /**
   * Resource update operation model.
   * @template Resource The resource model to update with replace.
   */
  @friendlyName("{name}ReplaceUpdate", Resource)
  model ResourceReplaceModel<Resource extends {}>
    is UpdateableProperties<DefaultKeyVisibility<Resource, Lifecycle.Read>>;
}
