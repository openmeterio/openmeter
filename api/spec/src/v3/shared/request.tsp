import "@typespec/rest";

namespace Shared;

@doc("{name} create request.", T)
@friendlyName("Create{name}Request", T)
@withVisibility(Lifecycle.Create)
model CreateRequest<T extends {}> is DefaultKeyVisibility<T, Lifecycle.Read>;

@doc("{name} upsert request.", T)
@friendlyName("Upsert{name}Request", T)
@withVisibility(Lifecycle.Create, Lifecycle.Update)
model UpsertRequest<T extends {}> is DefaultKeyVisibility<T, Lifecycle.Read>;

@doc("{name} update request.", T)
@friendlyName("Update{name}Request", T)
@withVisibility(Lifecycle.Update)
model UpdateRequest<T extends {}>
  is OptionalProperties<UpdateableProperties<DefaultKeyVisibility<
    T,
    Lifecycle.Read
  >>>;
