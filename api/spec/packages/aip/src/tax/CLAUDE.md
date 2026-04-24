# tax

<!-- archie:ai-start -->

> TypeSpec definitions for the Tax Codes v3 API domain: full CRUD for TaxCode resources that map OpenMeter app types to provider-specific tax codes. All endpoints are marked @extension(Shared.PrivateExtension, true) + UnstableExtension + InternalExtension — this is an internal/private API surface.

## Patterns

**Triple private/unstable/internal extension on all operations** — Every operation in TaxCodesOperations carries all three extensions: @extension(Shared.PrivateExtension, true), @extension(Shared.UnstableExtension, true), @extension(Shared.InternalExtension, true). Any new operation added here must carry all three. (`@extension(Shared.PrivateExtension, true) @extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) @post create(...): ...;`)
**Upsert over Create+Update pattern** — Tax codes use @put upsert (Shared.UpsertRequest<T> body, Shared.UpsertResponse<T> + Common.Gone response) rather than separate create/update endpoints. Use this pattern for resources that need idempotent writes. (`@put @operationId("upsert-tax-code") upsert(@path taxCodeId: Shared.ULID, @body tax_code: Shared.UpsertRequest<TaxCode>): Shared.UpsertResponse<TaxCode> | Common.Gone | Common.NotFound | Common.ErrorResponses;`)
**Cross-namespace import for app type reference** — codes.tsp imports `../apps/app.tsp` and references Apps.AppType directly. When a new model needs to reference another domain's types, import that domain's specific .tsp file (not its index) to minimize transitive dependencies. (`import "../apps/app.tsp"; model TaxCodeAppMapping { app_type: Apps.AppType; }`)
**ResourceWithKey base for key-addressable resources** — TaxCode spreads ...Shared.ResourceWithKey (not plain Resource) because tax codes are addressable by human-readable key. Use ResourceWithKey when the resource has a user-controlled lookup key. (`model TaxCode { ...Shared.ResourceWithKey; app_mappings: TaxCodeAppMapping[]; }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `codes.tsp` | TaxCode resource model and TaxCodeAppMapping value-object. TaxCode extends ResourceWithKey; app_mappings is a required array on all lifecycle operations. | app_mappings carries @visibility(Lifecycle.Read, Lifecycle.Update, Lifecycle.Create) — it is required on both create and update. Do not make it optional without considering backward compatibility. |
| `operations.tsp` | TaxCodesOperations interface: create, get, list (with include_deleted query param), upsert, delete. | List includes @query include_deleted?: boolean — a soft-delete pattern. The upsert path can return Common.Gone (410) if the resource was hard-deleted; handle this in the Go handler. |
| `index.tsp` | Barrel: imports codes.tsp then operations.tsp. | codes.tsp must be imported before operations.tsp since operations references TaxCode. |

## Anti-Patterns

- Adding a new endpoint without all three extensions (PrivateExtension, UnstableExtension, InternalExtension) — this is a private internal API
- Using Shared.Resource instead of Shared.ResourceWithKey — tax codes are key-addressable
- Splitting upsert into separate create+update endpoints — the domain uses idempotent @put upsert for this resource

## Decisions

- **Upsert (@put) instead of separate create/update for tax codes** — Tax codes are synced from external provider catalogs; idempotent upsert avoids 409 conflicts on repeated sync runs and simplifies the sync job.
- **All endpoints marked private+unstable+internal** — Tax code management is an operator-level function not exposed in the public SDK; the extensions exclude these endpoints from customer-facing SDK generation and documentation.

<!-- archie:ai-end -->
