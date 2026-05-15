# defaults

<!-- archie:ai-start -->

> TypeSpec definitions for the organization-level defaults subsystem: OrganizationDefaultTaxCodes model with get and update operations. Both operations are marked @extension(Shared.UnstableExtension, true) and @extension(Shared.InternalExtension, true) — this is an internal/unstable surface.

## Patterns

**Internal+Unstable extensions on all operations** — Every operation in this folder carries both @extension(Shared.UnstableExtension, true) and @extension(Shared.InternalExtension, true) to mark as internal/unstable in the generated OpenAPI. (`@get @extension(Shared.UnstableExtension, true) @extension(Shared.InternalExtension, true) get(): Shared.GetResponse<OrganizationDefaultTaxCodes> | Common.NotFound | Common.ErrorResponses;`)
**Shared.ResourceReference for nested references** — TaxCode references within the model use Shared.ResourceReference<Tax.TaxCode> rather than raw ULID fields, preserving typed linkage. (`invoicing_tax_code: Shared.ResourceReference<Tax.TaxCode>;`)
**@put with Shared.UpdateRequest for update** — The update operation uses @put with Shared.UpdateRequest (not @patch), consistent with the billing profile update pattern. (`@put update(@body body: Shared.UpdateRequest<OrganizationDefaultTaxCodes>): Shared.UpsertResponse<OrganizationDefaultTaxCodes> | ...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `taxcodes.tsp` | OrganizationDefaultTaxCodes model with invoicing_tax_code and credit_grant_tax_code fields plus read-only created_at/updated_at. | Model lacks Shared.Resource spread (no id field) — it is a singleton resource accessed by org context, not by ULID. |
| `operations.tsp` | OrganizationDefaultTaxCodesOperations interface with get and update only. Both marked internal+unstable. | Do not remove the UnstableExtension or InternalExtension annotations — this API is not yet public. |

## Anti-Patterns

- Removing the UnstableExtension or InternalExtension annotations — this is not a public API
- Adding a create or delete operation — the resource is a singleton provisioned automatically on org creation
- Using Shared.Resource spread — this singleton has no id field and is not identified by ULID

## Decisions

- **Singleton resource pattern without Shared.Resource spread** — OrganizationDefaultTaxCodes is a per-org singleton provisioned at org creation; it has no id and is not individually addressable, so it does not use the Shared.Resource (id/created_at/updated_at) spread.

<!-- archie:ai-end -->
