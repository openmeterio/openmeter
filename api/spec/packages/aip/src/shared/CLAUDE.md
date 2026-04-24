# shared

<!-- archie:ai-start -->

> Shared TypeSpec primitives (scalars, models, generics) reused across all aip/ domain namespaces. Acts as the type system foundation — every resource, filter, and response envelope for v3 is built from these definitions.

## Patterns

**Scalar with validation constraints** — New primitive types are TypeSpec scalars (not models) with @pattern, @minLength/@maxLength, @encode, @example, and @friendlyName decorators. See ULID, ResourceKey, CurrencyCode, DateTime. (`@pattern("^[A-Z]{3}$") @friendlyName("CurrencyCode") @minLength(3) @maxLength(3) scalar CurrencyCode extends string;`)
**Visibility lifecycle decorators on all fields** — Every model field uses @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update) to control which operations expose or accept the field. Read-only fields (id, created_at) carry only Lifecycle.Read. (`@visibility(Lifecycle.Read) id: ULID;`)
**Generic request/response envelopes via templates** — CreateRequest<T>, UpdateRequest<T>, UpsertRequest<T> in request.tsp and GetResponse<T>, CreateResponse<T>, CursorPaginatedResponse<T>, PagePaginatedResponse<T> in responses.tsp are parameterized generics. Domain models spread into these instead of redeclaring HTTP plumbing. (`model SubscriptionCreate { ...Shared.CreateRequest<OmitProperties<Subscription, "...">>;  }`)
**Resource base models via spread** — Resource (id, name, description, labels, timestamps), ResourceWithKey (adds key), ResourceImmutable (strips updated_at/deleted_at) are spread into domain models. Never redeclare these fields inline. (`model TaxCode { ...Shared.ResourceWithKey; ... }`)
**Filter models with logical AND/OR self-referential recursion** — QueryFilterString, QueryFilterInteger, QueryFilterFloat, QueryFilterNumeric, QueryFilterDateTime, QueryFilterBoolean all follow the pattern: operator fields + and?/or? arrays of self (up to 10 items). New filter types must follow this shape. (`model QueryFilterString { eq?: string; in?: string[]; and?: QueryFilterString[]; or?: QueryFilterString[]; }`)
**All exports via index.tsp barrel** — index.tsp imports all sibling .tsp files. Consumers import `../shared/index.tsp` — never individual files. New files added to shared/ must be added to index.tsp. (`import "./address.tsp"; import "./filters.tsp"; // etc in index.tsp`)
**Tag/description constants in consts.tsp** — API group names and descriptions are declared as `const` in consts.tsp and referenced by domain operations. Add new tag/description pairs here; do not inline strings in operations. (`const BillingTag = "OpenMeter Billing"; const BillingDescription = "...";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `properties.tsp` | Declares all shared scalar types (ULID, ResourceKey, DateTime, CurrencyCode, Numeric, ISO8601Duration, etc.) and value-object models (RecurringPeriod, ClosedPeriod, CurrencyAmount). | Adding a new scalar without @pattern/@example leads to missing OpenAPI validation. Numeric is a string scalar (arbitrary precision) — do not use float64. |
| `resource.tsp` | Resource, ResourceWithKey, ResourceImmutable base models that domain models spread into. ResourceReference<T> for cross-domain references. | Do not add domain-specific fields here. Spreading Resource into a model brings all lifecycle timestamps automatically. |
| `request.tsp` | Generic Create/Update/Upsert request wrappers that apply @withVisibility filtering. CreateRequestNested handles models with nested visibilities. | Use CreateRequestNested instead of CreateRequest when the model has nested structs with Lifecycle decorators — plain CreateRequest misses nested visibility filtering. |
| `responses.tsp` | HTTP response envelopes with hardcoded status codes (200/201/204). CursorPaginatedResponse and PagePaginatedResponse include @pageItems and meta. | Status codes are baked in — do not create ad-hoc response models with @statusCode; use these generics. |
| `filters.tsp` | Reusable query filter models for string, integer, float, numeric, boolean, datetime attributes, plus QueryFilterStringMap for map fields. | All filter models are self-referentially nested for AND/OR. Bounds are @minItems(1) @maxItems(10) for logical operators, @maxItems(100) for in/nin. |
| `consts.tsp` | String constants for all API tags, descriptions, and extension keys (UnstableExtension, InternalExtension, PrivateExtension). | Use Shared.UnstableExtension / InternalExtension / PrivateExtension via @extension() in operations; do not hard-code extension strings. |
| `index.tsp` | Barrel import — the single entry point for all shared types. | New .tsp files in this folder must be added here or they will be invisible to the rest of the spec. |

## Anti-Patterns

- Redeclaring id/name/created_at/updated_at/deleted_at on a domain model instead of spreading ...Shared.Resource or ...Shared.ResourceWithKey
- Using float64 or integer for monetary/quantity values — use Shared.Numeric (arbitrary-precision string scalar)
- Creating a filter model that does not include and?/or? self-references when logical composition is needed
- Defining HTTP status codes inline with @statusCode on domain-specific response models instead of using the generic envelopes in responses.tsp
- Adding a new .tsp file to shared/ without importing it in index.tsp

## Decisions

- **Scalars with inline validation constraints over Go-style validation in handlers** — TypeSpec scalars emit JSON Schema constraints directly into the OpenAPI spec, so all SDKs and the kin-openapi middleware validate automatically without duplicating rules.
- **Generic request/response templates instead of per-resource envelope models** — Eliminates drift between CRUD operations across 15+ domain namespaces; visibility filtering is applied once in the template and propagates everywhere.

## Example: Defining a new domain resource that extends shared base types and uses the CRUD envelope generics

```
import "../shared/index.tsp";

namespace MyDomain;

@friendlyName("MyResource")
model MyResource {
  ...Shared.ResourceWithKey;

  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Status")
  status: MyStatus;
}

interface MyResourceOperations {
  @post
// ...
```

<!-- archie:ai-end -->
