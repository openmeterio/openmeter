# currencies

<!-- archie:ai-start -->

> TypeSpec definitions for the v3 billing currencies API: fiat and custom currency models plus list/create operations, with a cost-bases/ child for per-currency rate records. All output compiles to api/v3/openapi.yaml and the Go/JS/Python SDKs via make gen-api; no hand-written Go or OpenAPI lives here.

## Patterns

**Discriminated union with envelope:none** — Currency is @discriminated with envelope 'none' and discriminatorPropertyName 'type'; new variants must extend the union, never add a flat polymorphic field. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Currency { fiat: CurrencyFiat, custom: CurrencyCustom }`)
**Generic base model CurrencyBase<T>** — Type-specific models spread CurrencyBase<CurrencyType.X> and add only their own fields; common fields (id, type, name, description, symbol) live only on CurrencyBase. (`model CurrencyCustom { ...CurrencyBase<CurrencyType.Custom>; @visibility(Lifecycle.Create, Lifecycle.Read) code: CurrencyCodeCustom; }`)
**@visibility on every field** — All fields carry explicit @visibility(Lifecycle.*); read-only system fields use Lifecycle.Read only, user-settable use Lifecycle.Create and/or Read. (`@visibility(Lifecycle.Read) id: Shared.ULID; @visibility(Lifecycle.Create, Lifecycle.Read) name: string;`)
**@friendlyName on every exported type** — Every model, enum, union, and scalar has @friendlyName("Billing<Name>") to stabilize generated SDK type names and prevent collisions. (`@friendlyName("BillingCurrencyCustom") model CurrencyCustom { ... }`)
**Stability @extension decorators on every operation** — Every operation carries @extension(Shared.InternalExtension, true) and @extension(Shared.UnstableExtension, true); omitting either breaks API maturity tracking. (`@extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) @get list(...)`)
**PagePaginationQuery + deepObject filter** — List operations spread ...Common.PagePaginationQuery and accept filter via @query(#{ style: "deepObject", explode: true }). (`list(...Common.PagePaginationQuery, @query(#{ style: "deepObject", explode: true }) filter?: ListCurrenciesParamsFilter)`)
**Models in currency.tsp, operations in operations.tsp** — currency.tsp defines models with no @typespec/http import; operations.tsp imports HTTP and declares the interfaces; index.tsp orders all imports including cost-bases/. (`// index.tsp: import "./currency.tsp"; import "./operations.tsp"; import "./cost-bases/operations.tsp";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `currency.tsp` | CurrencyType enum, Currency discriminated union, CurrencyBase<T> generic, CurrencyFiat, CurrencyCustom, CurrencyCodeCustom scalar. No HTTP imports. | Adding @get/@post or importing @typespec/http breaks model/operation separation; missing @friendlyName destabilizes SDK names; missing @visibility defaults to all lifecycle phases. |
| `operations.tsp` | CurrenciesOperations (list) and CurrenciesCustomOperations (create) with HTTP + stability decorators and ListCurrenciesParamsFilter. | Omitting @extension stability decorators breaks maturity tracking; using inline pagination params instead of ...Common.PagePaginationQuery causes contract drift. |
| `index.tsp` | Entry point importing currency.tsp, operations.tsp, and cost-bases/operations.tsp in order. | New .tsp files in this folder must be imported here or they are silently excluded from compilation. |

## Anti-Patterns

- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml instead of regenerating via make gen-api
- Declaring fields without @visibility — they default to all lifecycle phases and leak write-only/system fields into create payloads
- Adding operations without @extension(Shared.InternalExtension) and @extension(Shared.UnstableExtension)
- Using a @friendlyName that duplicates an existing model name — causes SDK type collisions
- Importing @typespec/http in currency.tsp — HTTP decorators belong only in operations.tsp

## Decisions

- **Currency is a discriminated union (envelope:none) rather than a flat polymorphic model** — Enables SDK discriminator deserialization without a wrapper envelope field, matching the fiat/custom split at the type level.
- **Models and operations split across currency.tsp and operations.tsp** — Keeps model definitions free of HTTP concerns so they can be imported without pulling in HTTP decorator dependencies.
- **Stability extensions on all operations** — The currencies API is internal/unstable; the decorators gate exposure in public OpenAPI outputs and SDK generation.

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
  @post create(@body body: Shared.CreateRequest<CurrencyToken>): Shared.CreateResponse<CurrencyToken> | Common.ErrorResponses;
// ...
```

<!-- archie:ai-end -->
