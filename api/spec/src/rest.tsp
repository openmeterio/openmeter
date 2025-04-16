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

namespace OpenMeter.Rest {
  /**
   * Resource create operation model.
   * @template Resource The resource model to create.
   */
  @friendlyName("{name}Create", Resource)
  @withVisibilityFilter(#{ all: #[Lifecycle.Create] })
  model ResourceCreateModel<Resource extends {}> {
    ...Resource;
  };

  /**
   * Resource update operation model.
   * @template Resource The resource model to partially update.
   */
  @friendlyName("{name}Update", Resource)
  @withVisibilityFilter(#{ all: #[Lifecycle.Update] })
  model ResourceUpdateModel<Resource extends {}> {
    ...Resource;
  };

  /**
   * Resource replace operation model.
   * @template Resource The resource model to update with replace.
   */
  @friendlyName("{name}Replace", Resource)
  @withVisibilityFilter(#{ any: #[Lifecycle.Create, Lifecycle.Update] })
  model ResourceReplaceModel<Resource extends {}> {
    ...Resource;
  };
}
