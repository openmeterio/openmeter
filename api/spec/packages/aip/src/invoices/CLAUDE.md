# invoices

<!-- archie:ai-start -->

> TypeSpec type definitions for invoice-related models (InvoiceNumber scalar, BillingParty, tax identity, addresses) shared across v3 billing operations. This folder contains only type definitions — invoice CRUD operations live in the billing/ namespace.

## Patterns

**Scalar types for domain-constrained strings** — Domain identifiers use scalar X extends string with @minLength/@maxLength/@example rather than plain string. (`@minLength(1) @maxLength(256) @friendlyName("BillingInvoiceNumber") scalar InvoiceNumber extends string;`)
**Explicit visibility on every mutable field** — Party model fields each declare @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) or a subset — no implicit all-lifecycle fields. id is Read only; key is Read+Create. (`@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) name?: string;`)
**friendlyName for SDK type naming** — Models use @friendlyName("Billing<Name>") to control generated SDK names and avoid collisions with Shared namespace types. (`@friendlyName("BillingParty") model BillingParty { ... }`)
**Intentional GOBL omissions documented in comments** — party.tsp carries a block comment listing GOBL party fields deliberately excluded (identities, people, inboxes, emails, websites, telephones, registration, logos, ext) with rationale. Do not silently add them. (`// Omitted: identities (maintained by apps), people, inboxes, emails, websites, telephones, registration, logos, ext`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `party.tsp` | BillingParty, BillingPartyAddresses, BillingPartyTaxIdentity — the supplier/customer party models used on invoices. | Many GOBL fields are intentionally omitted per the leading comment block — do not add them without design discussion. BillingPartyAddresses currently has only billing_address. |
| `invoice.tsp` | Defines only the InvoiceNumber scalar (namespace Invoices). Full invoice body models live in the billing/ namespace. | Do not add full invoice models here — they belong in billing/. This is a thin type definition only. |
| `index.tsp` | Barrel importing invoice.tsp and party.tsp. | No namespace declaration in this index.tsp — callers reference types by their fully qualified Invoices.* name. |

## Anti-Patterns

- Adding full invoice CRUD operations here — operations belong in the billing/ folder's operations.tsp.
- Adding GOBL fields documented as intentionally omitted (identities, logos, registration, ext, etc.) without design discussion.
- Defining party types without @friendlyName — generates SDK names that clash with Shared namespace types.
- Using plain string instead of the InvoiceNumber scalar for invoice-number fields.

## Decisions

- **BillingParty is kept intentionally minimal versus the full GOBL party spec.** — Many GOBL fields (identity codes, people, inboxes, websites, logos) are not needed for OpenMeter billing and are explicitly excluded to avoid API surface bloat.

## Example: Adding an optional address to BillingPartyAddresses

```
@friendlyName("BillingPartyAddresses")
model BillingPartyAddresses {
  billing_address: Shared.Address;
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  shipping_address?: Shared.Address;
}
```

<!-- archie:ai-end -->
