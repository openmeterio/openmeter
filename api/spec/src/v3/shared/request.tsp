namespace Shared;

@doc("{name} create request.", T)
@friendlyName("Create{name}Request", T)
@withVisibility(Lifecycle.Create)
model CreateRequest<T extends {}> is DefaultKeyVisibility<T, Lifecycle.Read>;
