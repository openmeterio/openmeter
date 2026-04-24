# cost-bases

<!-- archie:ai-start -->

> Defines TypeSpec models and REST operations for billing cost bases — the per-fiat-currency rate records attached to custom currencies. All output is compiled; no hand-written OpenAPI or Go here.

## Patterns

**Visibility-scoped fields** — Every field uses @visibility(Lifecycle.Create, Lifecycle.Read) or @visibility(Lifecycle.Read) to control which fields appear in create vs read responses. System-assigned fields (id, created_at) are Read-only. (`@visibility(Lifecycle.Read) id: Shared.ULID; @visibility(Lifecycle.Create, Lifecycle.Read) fiat_code: Shared.CurrencyCode;`)
**Shared generic wrappers** — Use Shared.CreateRequest<T> / Shared.CreateResponse<T> / Shared.PagePaginatedResponse<T> for request/response envelopes — never write ad-hoc body/response shapes. (`@body body: Shared.CreateRequest<CostBasis>): Shared.CreateResponse<CostBasis>`)
**deepObject filter query params** — Filtering parameters are grouped into a dedicated filter model and annotated @query(#{ style: 'deepObject', explode: true }) so URL params follow filter[field]=value convention. (`@query(#{ style: "deepObject", explode: true }) filter?: ListCostBasesParamsFilter`)
**Extension decorators for visibility/stability** — Every operation carries @extension(Shared.PrivateExtension, true), @extension(Shared.UnstableExtension, true), and/or @extension(Shared.InternalExtension, true) to annotate API maturity. Add these before shipping any new operation. (`@extension(Shared.PrivateExtension, true) @extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) @get`)
**Explicit @operationId and @summary** — Every operation must declare @operationId (kebab-case) and @summary for deterministic SDK method naming and OpenAPI documentation. (`@operationId("list-cost-bases") @summary("List cost bases")`)
**Namespace scoping** — Models and operations are declared inside 'namespace Currencies;' matching the parent domain — never use the root namespace or a different one. (`namespace Currencies; model CostBasis { ... }`)
**Pagination via Common.PagePaginationQuery spread** — List operations spread ...Common.PagePaginationQuery into the parameter list rather than declaring page/pageSize manually. (`get_cost_bases(@path currencyId: Shared.ULID, ...Common.PagePaginationQuery): Shared.PagePaginatedResponse<CostBasis>`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `cost-basis.tsp` | Defines the CostBasis model. Single source of truth for field names, types, and visibility. @friendlyName sets the generated Go/JS type name. | Omitting @visibility causes fields to appear in both create and read payloads unintentionally; @friendlyName must be globally unique across the spec. |
| `operations.tsp` | Declares the CurrenciesCustomCostBasesOperations interface with list and create endpoints. All imports, HTTP decorators, and using statements live here. | Missing 'using TypeSpec.Http;' causes @get/@post/@query to be unknown decorators. Error union must include both domain-specific errors (Common.NotFound) and the catch-all Common.ErrorResponses. |

## Anti-Patterns

- Hand-editing generated OpenAPI or Go files instead of modifying these .tsp sources and running make gen-api
- Declaring fields without @visibility — fields default to all lifecycle phases and leak write-only or system fields into create payloads
- Adding operations without the three @extension stability decorators — breaks internal API maturity tracking
- Writing ad-hoc pagination parameters instead of spreading ...Common.PagePaginationQuery
- Using a @friendlyName that duplicates an existing model name — causes SDK type collisions

## Decisions

- **Models and operations split across two files (cost-basis.tsp + operations.tsp)** — Keeps the domain model (fields, types) separate from HTTP routing concerns (decorators, error unions, query params), matching the pattern used by sibling folders under aip/src/.
- **All operations marked @extension(Shared.InternalExtension) / PrivateExtension / UnstableExtension** — Cost-basis endpoints are billing-internal and not yet stable; the extensions prevent them from appearing in public SDK docs and signal breaking-change freedom.

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
