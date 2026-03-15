# Legacy API (`packages/legacy`)

This package defines OpenMeter v1-v2 APIs and OpenMeter Cloud API using TypeSpec, following OpenMeter's existing conventions.

**Outputs:** `openapi.OpenMeter.yaml`, `openapi.OpenMeterCloud.yaml`

---

## Naming conventions

- **Models**: `PascalCase` (e.g. `BillingProfile`, `PlanPhase`)
- **Model properties**: `camelCase` (e.g. `createdAt`, `billingProfileId`)
- **Enum names**: `PascalCase`; enum member names: `PascalCase`; enum values: `camelCase` (e.g. `CreatedAt: "createdAt"`)
- **Operation IDs**: `camelCase` verb+noun (e.g. `createMeter`, `listBillingProfiles`)

---

## Use `@visibility` to control property exposure

`@visibility` controls which properties appear in which request/response shapes, allowing one model to serve multiple contexts.

- `Lifecycle.Read` — returned by any operation
- `Lifecycle.Create` — accepted in create (POST) operations
- `Lifecycle.Update` — accepted in update (PUT/PATCH) operations

```tsp
model Meter {
  @visibility(Lifecycle.Read)
  id: ULID;

  @visibility(Lifecycle.Read, Lifecycle.Create)
  slug: Key;

  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  groupBy?: Record<string>;
}
```

---

## Use Rest resource models for request bodies

Derive request body models from TypeSpec's built-in REST resource utilities rather than defining ad-hoc types. This ensures correct property filtering by visibility.

| HTTP method | Template                                                | Naming convention |
| ----------- | ------------------------------------------------------- | ----------------- |
| POST        | `TypeSpec.Rest.Resource.ResourceCreateModel<T>`         | `{Name}Create`    |
| PUT         | `TypeSpec.Rest.Resource.ResourceReplaceModel<T>`        | `{Name}Update`    |
| PATCH       | `TypeSpec.Rest.Resource.ResourceCreateOrUpdateModel<T>` | `{Name}Patch`     |

```tsp
@friendlyName("MeterCreate")
model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;

@friendlyName("MeterUpdate")
model MeterUpdate is TypeSpec.Rest.Resource.ResourceReplaceModel<Meter>;
```

Avoid names like `RequestBody`, `Input`, or `Payload` for CRUD operations.

---

## Use `@friendlyName` with a domain prefix

Every named model must have a `@friendlyName` with a short domain-scoped name. This name controls the generated OpenAPI schema name.

---

## Suppress linter warnings intentionally

When you need to deviate from a rule (e.g. preserving existing API values), use `#suppress` with a clear reason.

---

## Route and tag conventions

- Routes follow the pattern `/api/v1/{resource}` with explicit `@route` decorators.
- Use `@tag` to group operations in the generated spec (e.g. `@tag("Meters")`).
- Use `@sharedRoute` when multiple operation overloads share a path (e.g. JSON vs CSV responses).

---

## Add `@example` to models and operations

Include realistic `@example` values on models and properties to improve generated documentation and SDK ergonomics.
