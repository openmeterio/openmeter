# currencies

<!-- archie:ai-start -->

> TypeSpec definitions for the billing currencies API: fiat and custom currency models plus list/create operations. Compiles to v3 OpenAPI spec and SDKs; no hand-written Go or OpenAPI here.

## Patterns

**Discriminated union with envelope:none** — Currency is a @discriminated union with envelope:'none' and discriminatorPropertyName:'type'. New currency variants must follow this pattern. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Currency { fiat: CurrencyFiat, custom: CurrencyCustom }`)
**Generic base model with CurrencyType parameter** — CurrencyBase<T extends CurrencyType> holds common fields. Type-specific models spread CurrencyBase with the concrete type and add their own fields. (`model CurrencyFiat { ...CurrencyBase<CurrencyType.Fiat>; @visibility(Lifecycle.Read) code: Shared.CurrencyCode; }`)
**@visibility on every field** — All fields must carry explicit @visibility(Lifecycle.*) annotations. Read-only system fields use Lifecycle.Read only; create+read fields use both. (`@visibility(Lifecycle.Read) id: Shared.ULID; @visibility(Lifecycle.Create, Lifecycle.Read) name: string;`)
**@friendlyName on every model, enum, scalar** — Every exported model, enum, and scalar must have @friendlyName("Billing<Name>") to stabilize generated SDK type names. (`@friendlyName("BillingCurrencyCustom") model CurrencyCustom { ... }`)
**Three stability @extension decorators on every operation** — Every operation must carry @extension(Shared.InternalExtension, true), @extension(Shared.PrivateExtension, true), @extension(Shared.UnstableExtension, true) for API maturity tracking. (`@extension(Shared.PrivateExtension, true) @extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) @get list(...)`)
**PagePaginationQuery spread + deepObject filter** — List operations spread ...Common.PagePaginationQuery and pass filter via @query(#{ style: "deepObject", explode: true }). (`list(...Common.PagePaginationQuery, @query(#{ style: "deepObject", explode: true }) filter?: ListCurrenciesParamsFilter)`)
**Models in currency.tsp, HTTP operations in operations.tsp** — Model definitions live in currency.tsp (no HTTP imports). HTTP decorators (@get, @post, @path, @query) live only in operations.tsp which imports @typespec/http. (`currency.tsp: model CurrencyCustom { ... } | operations.tsp: interface CurrenciesCustomOperations { @post create(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `currency.tsp` | Defines CurrencyType enum, Currency union, CurrencyBase generic, CurrencyFiat, CurrencyCustom, and CurrencyCodeCustom scalar. No HTTP imports. | Adding @get/@post or importing @typespec/http here breaks separation of concerns. Missing @friendlyName causes unstable SDK names. |
| `operations.tsp` | Declares CurrenciesOperations (list) and CurrenciesCustomOperations (create) interfaces with HTTP decorators, stability extensions, and filter model. | Omitting stability @extension decorators breaks internal API maturity tracking. Using inline pagination params instead of ...Common.PagePaginationQuery causes drift. |
| `index.tsp` | Entry point that imports currency.tsp, operations.tsp, and cost-bases/operations.tsp in order. | New .tsp files in this folder must be imported here; otherwise they are silently excluded from compilation. |

## Anti-Patterns

- Hand-editing api/v3/api.gen.go or api/v3/openapi.yaml — always regenerate via `make gen-api`
- Declaring fields without @visibility — fields default to all lifecycle phases, leaking write-only or system fields into create payloads
- Adding operations without the three @extension stability decorators — breaks internal API maturity tracking
- Using a @friendlyName that duplicates an existing model name — causes SDK type collisions
- Importing @typespec/http in currency.tsp (model file) — HTTP decorators belong only in operations.tsp

## Decisions

- **Currency is a discriminated union (envelope:none) rather than a flat polymorphic model** — Enables SDK discriminator deserialization without a wrapper envelope field, matching the fiat/custom split at the type level.
- **Models and operations split across currency.tsp and operations.tsp** — Keeps model definitions free of HTTP concerns so they can be imported without pulling in HTTP decorator dependencies.
- **@extension(Shared.InternalExtension/PrivateExtension/UnstableExtension) on all operations** — Currencies API is internal/unstable; stability decorators gate exposure in public-facing OpenAPI outputs and SDK generation.

## Example: Adding a new custom currency variant with create operation

```
// currency.tsp
@friendlyName("BillingCurrencyToken")
model CurrencyToken {
  ...CurrencyBase<CurrencyType.Custom>;
  @visibility(Lifecycle.Create, Lifecycle.Read)
  code: CurrencyCodeCustom;
  @visibility(Lifecycle.Create, Lifecycle.Read)
  decimals: uint8;
}

// operations.tsp
interface CurrenciesTokenOperations {
  @extension(Shared.PrivateExtension, true)
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
// ...
```

<!-- archie:ai-end -->
