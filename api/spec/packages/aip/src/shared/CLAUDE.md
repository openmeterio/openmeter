# shared

<!-- archie:ai-start -->

> Shared TypeSpec primitives (scalars, models, generics) reused across all aip/ domain namespaces — the type system foundation for the v3 API. Every resource base, filter type, pagination envelope, and response wrapper for v3 is built here; changes propagate across all domains.

## Patterns

**Scalar with validation constraints** — New primitive types are TypeSpec scalars (not models) with @pattern, @minLength/@maxLength, @encode, @example, and @friendlyName decorators. ULID, ResourceKey, CurrencyCode, DateTime, ISO8601Duration, and Numeric all follow this pattern. (`@pattern("^[A-Z]{3}$") @friendlyName("CurrencyCode") @minLength(3) @maxLength(3) @example("USD") scalar CurrencyCode extends string;`)
**@visibility lifecycle decorators on all model fields** — Every field in every model uses @visibility(Lifecycle.Create, Lifecycle.Read, Lifecycle.Update) to control which operations expose or accept the field. Read-only fields (id, created_at, updated_at, deleted_at) carry only Lifecycle.Read. (`@visibility(Lifecycle.Read) id: ULID;
@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) name: string;`)
**Generic request/response envelopes via templates** — CreateRequest<T>, UpdateRequest<T>, UpsertRequest<T>, GetResponse<T>, CreateResponse<T>, CursorPaginatedResponse<T>, and PagePaginatedResponse<T> are parameterized generics. Domain models spread into these instead of redeclaring HTTP status codes or pagination plumbing. (`// Domain uses CreateRequest envelope:
model SubscriptionCreate { ...Shared.CreateRequest<OmitProperties<Subscription, "...">>; }`)
**Resource base models spread into domain models** — Resource (id, name, description, labels, timestamps), ResourceWithKey (adds key), ResourceImmutable (strips updated_at/deleted_at) are the only base models. Domain models spread one of these instead of redeclaring any of those fields inline. (`model TaxCode { ...Shared.ResourceWithKey; ... }`)
**Filter models with self-referential AND/OR recursion** — All query filter models (QueryFilterString, QueryFilterInteger, QueryFilterNumeric, QueryFilterDateTime, QueryFilterBoolean) include and?/or? arrays of themselves bounded by @minItems(1) @maxItems(10). New filter types must follow this shape. (`model QueryFilterString { eq?: string; in?: string[]; @maxItems(10) and?: QueryFilterString[]; @maxItems(10) or?: QueryFilterString[]; }`)
**index.tsp barrel — single import entry point** — All sibling .tsp files are imported by index.tsp. Consumers import only ../shared/index.tsp — never individual files. New files added to shared/ must be added to index.tsp. (`// In consumer domain:
import "../shared/index.tsp";
// NOT: import "../shared/filters.tsp";`)
**Extension constants in consts.tsp** — API group names, descriptions, and extension keys (UnstableExtension, InternalExtension, PrivateExtension) are declared as `const` in consts.tsp. Operations reference these via @extension(Shared.UnstableExtension, true) — never inline the string. (`const UnstableExtension = "x-unstable";
// In operations:
@extension(Shared.UnstableExtension, true)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `properties.tsp` | All shared scalar types (ULID, ResourceKey, DateTime, CurrencyCode, Numeric, ISO8601Duration) and value-object models (RecurringPeriod, ClosedPeriod, CurrencyAmount). | Numeric is a string scalar (arbitrary-precision) — do not substitute float64 or integer for monetary/quantity values. DateTime uses @encode(rfc3339) — not a plain string. |
| `resource.tsp` | Resource, ResourceWithKey, ResourceImmutable base models and ResourceReference<T> for cross-domain references. | Spreading Resource brings all lifecycle timestamps automatically. Do not add domain-specific fields to these base models. ResourceReference<T> contains only id: ULID. |
| `request.tsp` | Generic Create/Update/Upsert request wrappers with @withVisibility filtering. CreateRequestNested handles models with nested visibility decorators. | Use CreateRequestNested instead of CreateRequest when the model has nested structs with Lifecycle decorators — plain CreateRequest misses nested visibility filtering. |
| `responses.tsp` | HTTP response envelopes with hardcoded status codes (GetResponse=200, CreateResponse=201, UpsertResponse=200, DeleteResponse=204). CursorPaginatedResponse and PagePaginatedResponse include @pageItems data array and meta. | Status codes are baked into the generics — do not create ad-hoc response models with @statusCode; always use these generics. |
| `filters.tsp` | Reusable query filter models for string, integer, float, numeric, boolean, datetime attributes. QueryFilterStringMapItem extends QueryFilterString with an exists? field for map entries. | in/nin arrays: @maxItems(100). and/or arrays: @maxItems(10). All filters are self-referentially nested for logical composition. |
| `consts.tsp` | String constants for all API tags, descriptions (MetersTag, BillingTag, etc.) and extension keys (UnstableExtension, InternalExtension, PrivateExtension). | Add new tag/description pairs here; do not inline strings in operations files. |

## Anti-Patterns

- Redeclaring id/name/created_at/updated_at/deleted_at on a domain model — spread ...Shared.Resource or ...Shared.ResourceWithKey instead
- Using float64 or integer for monetary/quantity values — use Shared.Numeric (arbitrary-precision string scalar)
- Creating a filter model without and?/or? self-references when logical composition is needed
- Defining HTTP status codes inline with @statusCode on domain-specific response models — use the generic envelopes in responses.tsp
- Adding a new .tsp file to shared/ without importing it in index.tsp

## Decisions

- **Scalars with inline validation constraints over Go-side validation in handlers** — TypeSpec scalars emit JSON Schema constraints directly into the OpenAPI spec, so all SDKs and the kin-openapi middleware validate automatically without duplicating rules in Go.
- **Generic request/response templates instead of per-resource envelope models** — Eliminates drift between CRUD operations across 15+ domain namespaces; visibility filtering is applied once in the template and propagates everywhere.

## Example: Defining a new domain resource extending shared base types with CRUD envelopes

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
  @post @operationId("create-my-resource")
// ...
```

<!-- archie:ai-end -->
