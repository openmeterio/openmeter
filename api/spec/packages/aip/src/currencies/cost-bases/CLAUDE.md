# cost-bases

<!-- archie:ai-start -->

> Defines TypeSpec models and REST operations for billing cost bases — per-fiat-currency rate records attached to custom currencies. All output is compiled via `make gen-api`; no hand-written OpenAPI or Go here.

## Patterns

**Visibility-scoped fields** — Every field must declare @visibility(Lifecycle.Create, Lifecycle.Read) or @visibility(Lifecycle.Read). System-assigned fields (id, created_at) are Read-only. Omitting @visibility leaks system fields into create payloads. (`@visibility(Lifecycle.Read) id: Shared.ULID; @visibility(Lifecycle.Create, Lifecycle.Read) fiat_code: Shared.CurrencyCode;`)
**Shared generic wrappers** — Use Shared.CreateRequest<T> / Shared.CreateResponse<T> / Shared.PagePaginatedResponse<T> for all request/response envelopes — never write ad-hoc body or response shapes. (`@body body: Shared.CreateRequest<CostBasis>): Shared.CreateResponse<CostBasis>`)
**deepObject filter query params** — Filter parameters must be grouped into a dedicated filter model annotated @query(#{ style: 'deepObject', explode: true }) so URL params follow filter[field]=value convention. (`@query(#{ style: "deepObject", explode: true }) filter?: ListCostBasesParamsFilter`)
**Extension decorators for stability** — Every operation must carry @extension(Shared.UnstableExtension, true) and @extension(Shared.InternalExtension, true) (and optionally @extension(Shared.PrivateExtension, true)) before shipping. Missing these breaks internal API maturity tracking. (`@extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) @get`)
**Explicit @operationId and @summary** — Every operation must declare @operationId (kebab-case) and @summary for deterministic SDK method naming and OpenAPI documentation. (`@operationId("list-cost-bases") @summary("List cost bases")`)
**Namespace scoping** — All models and operations must be declared inside `namespace Currencies;` matching the parent domain. Never use the root namespace or a different one. (`namespace Currencies; model CostBasis { ... }`)
**Pagination via Common.PagePaginationQuery spread** — List operations spread ...Common.PagePaginationQuery into the parameter list rather than declaring page/pageSize manually. (`get_cost_bases(@path currencyId: Shared.ULID, ...Common.PagePaginationQuery): Shared.PagePaginatedResponse<CostBasis>`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `cost-basis.tsp` | Defines the CostBasis model — single source of truth for field names, types, and visibility. @friendlyName sets the generated Go/JS type name. | Omitting @visibility causes fields to appear in both create and read payloads unintentionally. @friendlyName must be globally unique across the spec — duplicates cause SDK type collisions. |
| `operations.tsp` | Declares the CurrenciesCustomCostBasesOperations interface with list and create endpoints. All imports, HTTP decorators, and using statements live here. | Missing 'using TypeSpec.Http;' makes @get/@post/@query unknown decorators. Error union must include both domain-specific errors (Common.NotFound) and the catch-all Common.ErrorResponses. |

## Anti-Patterns

- Hand-editing generated OpenAPI or Go files instead of modifying these .tsp sources and running make gen-api
- Declaring fields without @visibility — fields default to all lifecycle phases and leak write-only or system fields into create payloads
- Adding operations without the stability @extension decorators — breaks internal API maturity tracking
- Writing ad-hoc pagination parameters instead of spreading ...Common.PagePaginationQuery
- Using a @friendlyName that duplicates an existing model name — causes SDK type collisions

## Decisions

- **Models and operations split across two files (cost-basis.tsp + operations.tsp)** — Keeps the domain model (fields, types) separate from HTTP routing concerns (decorators, error unions, query params), matching the pattern used by sibling folders under aip/src/.
- **All operations marked @extension(Shared.InternalExtension) and @extension(Shared.UnstableExtension)** — Cost-basis endpoints are billing-internal and not yet stable; the extensions prevent them from appearing in public SDK docs and signal breaking-change freedom.

## Example: Adding a new read-only field to CostBasis (e.g. updated_at)

```
// In cost-basis.tsp, inside model CostBasis:
/**
 * An ISO-8601 timestamp of the last update.
 */
@visibility(Lifecycle.Read)
updated_at?: Shared.DateTime;
// Then run: make gen-api && make generate
```

<!-- archie:ai-end -->
