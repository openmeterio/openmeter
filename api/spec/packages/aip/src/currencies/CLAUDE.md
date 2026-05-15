# currencies

<!-- archie:ai-start -->

> TypeSpec definitions for the billing currencies API: fiat and custom currency models plus list/create operations. All output compiles to v3 OpenAPI spec and SDKs via `make gen-api`; no hand-written Go or OpenAPI here.

## Patterns

**Discriminated union with envelope:none** — Currency is a @discriminated union with envelope:'none' and discriminatorPropertyName:'type'. New currency variants must extend this union — never add a flat polymorphic field. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Currency { fiat: CurrencyFiat, custom: CurrencyCustom }`)
**Generic base model CurrencyBase<T>** — Type-specific models spread CurrencyBase<CurrencyType.X> and add only their own fields. Common fields (id, type, name, description, symbol) live exclusively on CurrencyBase. (`model CurrencyCustom { ...CurrencyBase<CurrencyType.Custom>; @visibility(Lifecycle.Create, Lifecycle.Read) code: CurrencyCodeCustom; }`)
**@visibility on every field** — All fields carry explicit @visibility(Lifecycle.*) annotations. Read-only system fields use Lifecycle.Read only; user-settable fields use Lifecycle.Create and/or Lifecycle.Read. (`@visibility(Lifecycle.Read) id: Shared.ULID; @visibility(Lifecycle.Create, Lifecycle.Read) name: string;`)
**@friendlyName on every exported type** — Every model, enum, union, and scalar must have @friendlyName("Billing<Name>") to stabilize generated SDK type names and prevent collisions. (`@friendlyName("BillingCurrencyCustom") model CurrencyCustom { ... }`)
**Stability @extension decorators on every operation** — Every operation carries @extension(Shared.InternalExtension, true) and @extension(Shared.UnstableExtension, true). Omitting either breaks API maturity tracking in the generated OpenAPI. (`@extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) @get list(...)`)
**PagePaginationQuery spread + deepObject filter** — List operations spread ...Common.PagePaginationQuery for pagination and accept filter via @query(#{ style: "deepObject", explode: true }). (`list(...Common.PagePaginationQuery, @query(#{ style: "deepObject", explode: true }) filter?: ListCurrenciesParamsFilter)`)
**Models in currency.tsp, HTTP operations in operations.tsp** — currency.tsp defines all models/enums/scalars with no HTTP imports. operations.tsp imports @typespec/http and declares all interface operations. cost-bases/ is a child folder imported via index.tsp. (`// currency.tsp: no HTTP imports | operations.tsp: using TypeSpec.Http; interface CurrenciesCustomOperations { @post create(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `currency.tsp` | Defines CurrencyType enum, Currency discriminated union, CurrencyBase<T> generic, CurrencyFiat, CurrencyCustom, and CurrencyCodeCustom scalar. No HTTP imports. | Adding @get/@post or importing @typespec/http breaks the model/operation separation. Missing @friendlyName causes unstable SDK names. Missing @visibility defaults all lifecycle phases. |
| `operations.tsp` | Declares CurrenciesOperations (list) and CurrenciesCustomOperations (create) with HTTP decorators, stability extensions, and ListCurrenciesParamsFilter model. | Omitting @extension stability decorators breaks internal API maturity tracking. Using inline pagination params instead of ...Common.PagePaginationQuery causes drift from the standard pagination contract. |
| `index.tsp` | Entry point that imports currency.tsp, operations.tsp, and cost-bases/operations.tsp in order. | New .tsp files added to this folder must be imported here; silently excluded from compilation otherwise. |

## Anti-Patterns

- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml — always regenerate via `make gen-api`
- Declaring fields without @visibility — fields default to all lifecycle phases, leaking write-only or system fields into create payloads
- Adding operations without @extension(Shared.InternalExtension) and @extension(Shared.UnstableExtension) — breaks internal API maturity tracking
- Using a @friendlyName that duplicates an existing model name — causes SDK type collisions
- Importing @typespec/http in currency.tsp (model file) — HTTP decorators belong only in operations.tsp

## Decisions

- **Currency is a discriminated union (envelope:none) rather than a flat polymorphic model** — Enables SDK discriminator deserialization without a wrapper envelope field, matching the fiat/custom split at the type level.
- **Models and operations split across currency.tsp and operations.tsp** — Keeps model definitions free of HTTP concerns so they can be imported without pulling in HTTP decorator dependencies.
- **@extension(Shared.InternalExtension/UnstableExtension) on all operations** — Currencies API is internal/unstable; stability decorators gate exposure in public-facing OpenAPI outputs and SDK generation.

## Example: Adding a new custom currency variant with a create operation

```
// currency.tsp — model only, no HTTP imports
@friendlyName("BillingCurrencyToken")
model CurrencyToken {
  ...CurrencyBase<CurrencyType.Custom>;
  @visibility(Lifecycle.Create, Lifecycle.Read)
  code: CurrencyCodeCustom;
  @visibility(Lifecycle.Create, Lifecycle.Read)
  decimals: uint8;
}

// operations.tsp — HTTP decorators only here
interface CurrenciesTokenOperations {
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
// ...
```

<!-- archie:ai-end -->
