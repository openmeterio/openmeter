import "@typespec/rest";
import "../../lib/decorators.js";

namespace Shared;

extern dec withNullableOptionalProperties(target: Reflection.Model);

@doc("{name} create request.", T)
@friendlyName("Create{name}Request", T)
@withVisibility(Lifecycle.Create)
model CreateRequest<T extends {}> is DefaultKeyVisibility<T, Lifecycle.Read>;

// Supports visibility filtering for models with nested fields.
@doc("{name} create request.", T)
@friendlyName("Create{name}Request", T)
model CreateRequestNested<T extends {}> is FilterVisibility<
  DefaultKeyVisibility<T, Lifecycle.Read>,
  #{ all: #[Lifecycle.Create] },
  "Create{name}"
>;

@doc("{name} upsert request.", T)
@friendlyName("Upsert{name}Request", T)
@withVisibility(Lifecycle.Create, Lifecycle.Update)
model UpsertRequest<T extends {}> is DefaultKeyVisibility<T, Lifecycle.Read>;

// Shares the Upsert{name}Request friendly name with UpsertRequest so switching
// a resource between the two keeps generated type names stable; passing the
// same T to both templates is therefore a generation-time name collision — a
// model must use one or the other.
@doc(
  "{name} upsert request. The request replaces the stored state: omitted optional fields are unset, and an explicit `null` is equivalent to omission.",
  T
)
@friendlyName("Upsert{name}Request", T)
@withVisibility(Lifecycle.Create, Lifecycle.Update)
@withNullableOptionalProperties
model UpsertNullableRequest<T extends {}> is DefaultKeyVisibility<
  T,
  Lifecycle.Read
>;

@doc("{name} update request.", T)
@friendlyName("Update{name}Request", T)
@withVisibility(Lifecycle.Update)
model UpdateRequest<T extends {}>
  is OptionalProperties<UpdateableProperties<DefaultKeyVisibility<
    T,
    Lifecycle.Read
  >>>;

// Supports visibility filtering for models with nested fields.
@doc("{name} update request.", T)
@friendlyName("Update{name}Request", T)
model UpdateRequestNested<T extends {}> is FilterVisibility<
  DefaultKeyVisibility<T, Lifecycle.Read>,
  #{ all: #[Lifecycle.Update] },
  "Update{name}"
>;
