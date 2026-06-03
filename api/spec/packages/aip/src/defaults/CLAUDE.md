# defaults

<!-- archie:ai-start -->

> v3 (AIP) TypeSpec for the organization-level defaults subsystem: the OrganizationDefaultTaxCodes singleton model plus get/update operations. Marked internal+unstable — this is not yet a public API surface.

## Patterns

**Internal+Unstable extensions on every operation** — Each operation carries both @extension(Shared.UnstableExtension, true) and @extension(Shared.InternalExtension, true) to mark it internal/unstable in the generated OpenAPI. (`@get @extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) get(): Shared.GetResponse<OrganizationDefaultTaxCodes> | Common.NotFound | Common.ErrorResponses;`)
**Shared.ResourceReference for typed links** — TaxCode references use Shared.ResourceReference<Tax.TaxCode> rather than raw ULID fields to preserve typed linkage. (`invoicing_tax_code: Shared.ResourceReference<Tax.TaxCode>;`)
**@put with Shared.UpdateRequest for update** — The update operation uses @put with Shared.UpdateRequest (not @patch), consistent with the billing-profile update pattern, returning Shared.UpsertResponse. (`@put update(@body body: Shared.UpdateRequest<OrganizationDefaultTaxCodes>): Shared.UpsertResponse<OrganizationDefaultTaxCodes> | Common.NotFound | Common.ErrorResponses;`)
**Read-only timestamps without Shared.Resource spread** — created_at/updated_at are declared directly with @visibility(Lifecycle.Read); the model intentionally omits the Shared.Resource (id) spread because it is a singleton. (`@visibility(Lifecycle.Read) created_at: Shared.DateTime;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `taxcodes.tsp` | OrganizationDefaultTaxCodes model with invoicing_tax_code, credit_grant_tax_code, and read-only created_at/updated_at. | Model lacks the Shared.Resource spread (no id) — it is a per-org singleton accessed by org context, not by ULID. |
| `operations.tsp` | OrganizationDefaultTaxCodesOperations interface with get and update only, both internal+unstable. | Do not remove UnstableExtension/InternalExtension annotations — this API is not yet public. namespace Defaults. |
| `index.tsp` | Barrel import re-exporting taxcodes.tsp and operations.tsp. | Every new .tsp file in this folder must be imported here. |

## Anti-Patterns

- Removing the UnstableExtension or InternalExtension annotations — this is not a public API
- Adding a create or delete operation — the resource is a singleton provisioned automatically at org creation
- Using the Shared.Resource spread — this singleton has no id and is not identified by ULID

## Decisions

- **Singleton resource pattern without the Shared.Resource spread** — OrganizationDefaultTaxCodes is a per-org singleton provisioned at org creation; it has no id and is not individually addressable, so it skips the id/created_at/updated_at Resource spread.

## Example: Singleton get operation marked internal+unstable

```
@get
@operationId("get-organization-default-tax-codes")
@summary("Get organization default tax codes")
@extension(Shared.UnstableExtension, true)
@extension(Shared.InternalExtension, true)
get(): Shared.GetResponse<OrganizationDefaultTaxCodes> | Common.NotFound | Common.ErrorResponses;
```

<!-- archie:ai-end -->
