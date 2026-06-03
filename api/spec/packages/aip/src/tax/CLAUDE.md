# tax

<!-- archie:ai-start -->

> TypeSpec definitions for the Tax Codes v3 API domain: full CRUD for TaxCode resources mapping OpenMeter app types (Stripe, Sandbox, etc.) to provider-specific tax codes. An internal/private API surface — all endpoints carry PrivateExtension, UnstableExtension, and InternalExtension.

## Patterns

**Triple private/unstable/internal extension on all operations** — Every operation in TaxCodesOperations carries @extension(Shared.PrivateExtension, true), @extension(Shared.UnstableExtension, true), and @extension(Shared.InternalExtension, true). New operations must carry all three. (`@post @operationId("create-tax-code") @extension(Shared.InternalExtension, true) @extension(Shared.UnstableExtension, true) create(@body tax_code: Shared.CreateRequest<TaxCode>): Shared.CreateResponse<TaxCode> | Common.ErrorResponses;`)
**Upsert (@put) over separate create+update for sync-friendly resources** — Tax codes use @put upsert (Shared.UpsertRequest<T> body, Shared.UpsertResponse<T> + Common.Gone) rather than separate create/update, because tax codes are synced idempotently from external provider catalogs. (`@put @operationId("upsert-tax-code") upsert(@path taxCodeId: Shared.ULID, @body tax_code: Shared.UpsertRequest<TaxCode>): Shared.UpsertResponse<TaxCode> | Common.Gone | Common.NotFound | Common.ErrorResponses;`)
**Cross-namespace import of a specific .tsp file (not index)** — codes.tsp imports only ../apps/app.tsp (not ../apps/index.tsp) to reference Apps.AppType. Import the specific .tsp file to minimize transitive dependencies. (`import "../apps/app.tsp";
model TaxCodeAppMapping { app_type: Apps.AppType; tax_code: string; }`)
**ResourceWithKey for key-addressable resources** — TaxCode spreads ...Shared.ResourceWithKey (not plain Resource) because tax codes are addressable by human-readable key in addition to ULID. (`model TaxCode { ...Shared.ResourceWithKey; app_mappings: TaxCodeAppMapping[]; }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `codes.tsp` | TaxCode resource model and TaxCodeAppMapping value-object; TaxCodeReference extends Shared.ResourceReference<TaxCode>. | app_mappings carries @visibility(Lifecycle.Read, Lifecycle.Update, Lifecycle.Create) — required on both create and update. Do not make it optional without backward-compatibility analysis. |
| `operations.tsp` | TaxCodesOperations interface: create, get, list (with include_deleted query param), upsert, delete. | List includes @query include_deleted?: boolean (soft-delete). Upsert can return Common.Gone (410) if the resource was hard-deleted — the Go handler must handle this. |
| `index.tsp` | Barrel: imports codes.tsp then operations.tsp in dependency order. | codes.tsp must be imported before operations.tsp since operations references TaxCode. |

## Anti-Patterns

- Adding a new endpoint without all three extensions (Private/Unstable/Internal) — this is a private internal API
- Using Shared.Resource instead of Shared.ResourceWithKey — tax codes are key-addressable
- Splitting upsert into separate create+update endpoints — idempotent upsert is required for the sync job pattern
- Importing ../apps/index.tsp instead of ../apps/app.tsp — import only the specific file containing the needed type

## Decisions

- **Upsert (@put) instead of separate create/update for tax codes** — Tax codes are synced from external provider catalogs; idempotent upsert avoids 409 conflicts on repeated sync runs and simplifies the sync job.
- **All endpoints marked private+unstable+internal** — Tax code management is an operator-level function not exposed in the public SDK; the extensions exclude these endpoints from customer-facing SDK generation and docs.

## Example: Adding a new TaxCode field following existing visibility patterns

```
// In codes.tsp:
model TaxCode {
  ...Shared.ResourceWithKey;
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Category")
  category?: string;
  @visibility(Lifecycle.Read, Lifecycle.Update, Lifecycle.Create)
  @summary("App type to tax code mappings")
  app_mappings: TaxCodeAppMapping[];
}
```

<!-- archie:ai-end -->
