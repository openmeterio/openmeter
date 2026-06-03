# shared

<!-- archie:ai-start -->

> Shared TypeSpec primitives (scalars, base models, generic request/response envelopes, filters) reused across all aip/ v3 domain namespaces — the type-system foundation for the v3 API. Changes here propagate across every domain.

## Patterns

**Scalar with validation constraints** — New primitive types are TypeSpec scalars (not models) with @pattern, @minLength/@maxLength, @encode, @example, @friendlyName. ULID, ResourceKey, CurrencyCode, DateTime, ISO8601Duration, Numeric follow this. (`@pattern("^[A-Z]{3}$") @friendlyName("CurrencyCode") @minLength(3) @maxLength(3) @example("USD") scalar CurrencyCode extends string;`)
**@visibility lifecycle decorators on all model fields** — Every field uses @visibility(Lifecycle.Create/Read/Update). Read-only fields (id, created_at, updated_at, deleted_at) carry only Lifecycle.Read. (`@visibility(Lifecycle.Read) id: ULID;
@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) name: string;`)
**Generic request/response envelopes via templates** — CreateRequest<T>, UpdateRequest<T>, UpsertRequest<T>, GetResponse<T>, CreateResponse<T>, CursorPaginatedResponse<T>, PagePaginatedResponse<T> are parameterized generics. Domain models spread into these instead of redeclaring status codes or pagination plumbing. (`model SubscriptionCreate { ...Shared.CreateRequest<OmitProperties<Subscription, "...">>; }`)
**Resource base models spread into domain models** — Resource (id, name, description, labels, timestamps), ResourceWithKey (adds key), ResourceImmutable (strips updated_at/deleted_at) are the only base models. Domain models spread one rather than redeclaring those fields. (`model TaxCode { ...Shared.ResourceWithKey; ... }`)
**Filter models with self-referential AND/OR recursion** — Query filter models (QueryFilterString/Integer/Numeric/DateTime/Boolean) include and?/or? arrays of themselves; in/nin bounded by @maxItems(100), and/or by @maxItems(10). New filter types follow this shape. (`model QueryFilterString { eq?: string; in?: string[]; @maxItems(10) and?: QueryFilterString[]; @maxItems(10) or?: QueryFilterString[]; }`)
**index.tsp barrel — single import entry point** — All sibling .tsp files are imported by index.tsp; consumers import only ../shared/index.tsp, never individual files. New files must be added to index.tsp. (`import "../shared/index.tsp";  // NOT: import "../shared/filters.tsp";`)
**Extension constants in consts.tsp** — API tag names, descriptions, and extension keys (UnstableExtension, InternalExtension, PrivateExtension) are const in consts.tsp; operations reference Shared.<Const>, never inline the string. (`const UnstableExtension = "x-unstable";  // @extension(Shared.UnstableExtension, true)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `properties.tsp` | Shared scalar types (ULID, ResourceKey, DateTime, CurrencyCode, Numeric, ISO8601Duration) and value-object models (RecurringPeriod, ClosedPeriod, CurrencyAmount). | Numeric is a string scalar (arbitrary-precision) — never substitute float64/integer for monetary/quantity values. DateTime uses @encode(rfc3339), not a plain string. |
| `resource.tsp` | Resource, ResourceWithKey, ResourceImmutable base models and ResourceReference<T> for cross-domain references. | Spreading Resource brings all lifecycle timestamps automatically. Do not add domain-specific fields to base models. ResourceReference<T> contains only id: ULID. |
| `request.tsp` | Generic Create/Update/Upsert request wrappers with @withVisibility filtering; CreateRequestNested handles nested-visibility models. | Use CreateRequestNested (not plain CreateRequest) when the model has nested structs with Lifecycle decorators — plain CreateRequest misses nested visibility filtering. |
| `responses.tsp` | HTTP response envelopes with hardcoded status codes (GetResponse=200, CreateResponse=201, UpsertResponse=200, DeleteResponse=204); CursorPaginatedResponse/PagePaginatedResponse with @pageItems data array and meta. | Status codes are baked into the generics — do not create ad-hoc response models with @statusCode; always use these generics. |
| `filters.tsp` | Reusable query filter models for string/integer/float/numeric/boolean/datetime; QueryFilterStringMapItem extends QueryFilterString with exists? for map entries. | in/nin arrays @maxItems(100); and/or arrays @maxItems(10). All filters are self-referentially nested for logical composition. |
| `consts.tsp` | String constants for all API tags/descriptions (MetersTag, BillingTag, etc.) and extension keys (UnstableExtension, InternalExtension, PrivateExtension). | Add new tag/description pairs here; do not inline strings in operations files. |

## Anti-Patterns

- Redeclaring id/name/created_at/updated_at/deleted_at on a domain model — spread ...Shared.Resource or ...Shared.ResourceWithKey
- Using float64 or integer for monetary/quantity values — use Shared.Numeric
- Creating a filter model without and?/or? self-references when logical composition is needed
- Defining HTTP status codes inline with @statusCode on domain response models — use the generic envelopes
- Adding a new .tsp file to shared/ without importing it in index.tsp

## Decisions

- **Scalars with inline validation constraints over Go-side validation in handlers** — TypeSpec scalars emit JSON Schema constraints into OpenAPI, so all SDKs and kin-openapi/oasmiddleware validate automatically without duplicating rules in Go.
- **Generic request/response templates instead of per-resource envelope models** — Eliminates CRUD drift across 15+ domain namespaces; visibility filtering is applied once in the template and propagates everywhere.

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
  @post @operationId("create-my-resource") create(@body body: Shared.CreateRequest<MyResource>): Shared.CreateResponse<MyResource>;
}
```

<!-- archie:ai-end -->
