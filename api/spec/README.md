# TypeSpec best practices

## Use `@visibility` decorator

Visibility is a language feature that allows you to share a model between multiple operations and define in which contexts properties of the model are “visible.” Visibility is a very powerful feature that allows you to define different “views” of a model within different operations or contexts.

- `Lifecycle.Read`: output of any operation
- `Lifecycle.Create`: input to operations that create an entity
- `Lifecycle.Update`: input to operations that update data

Use the `@visibility` decorator to control the visibility of the properties in the generated OpenAPI specification.

## Use `Rest` (`"@typespec/rest"`) models to create request body types

- _POST_: `Rest.Resource.ResourceCreateModel<T>`
  - Fields with `Lifecycle.Create` visibility
  - The model name is `{name}Create`
- _PUT_: `Rest.Resource.ResourceReplaceModel<T>` (custom in `rest.tsp`)
  - Fields without `Lifecycle.Read` visibility
  - The model name is `{name}ReplaceUpdate`
- _PATCH_: `Rest.Resource.ResourceCreateOrUpdateModel<T>`
  - Fields without `Lifecycle.Read` visibility and all optional
  - The model name is `{name}Update`

Follow the naming convention with custom, CRUD type operations. Avoid names, like `RequestBody`, `Input` (this can be used for non CRUD operations).

## Use of the `@friendlyName` decorator

Use package prefix for the friendly name, like `Plan`, `PlanPhase`, `PlanStatus`.
