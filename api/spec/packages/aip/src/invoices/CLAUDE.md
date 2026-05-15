# invoices

<!-- archie:ai-start -->

> TypeSpec type definitions for invoice-related models (InvoiceNumber scalar, BillingParty, tax identity, addresses) shared across billing operations in the v3 API. This folder contains only type definitions — invoice CRUD operations live in the billing/ namespace.

## Patterns

**Scalar types for domain-constrained strings** — Use scalar X extends string with @minLength/@maxLength/@example for strongly-typed domain identifiers like InvoiceNumber rather than using plain string. (`@example("INV-2024-01-01-01")
@minLength(1)
@maxLength(256)
@friendlyName("BillingInvoiceNumber")
scalar InvoiceNumber extends string;`)
**Explicit visibility on every mutable field** — Every field on party models explicitly declares @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) or a subset — no implicit all-lifecycle fields. (`@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
name?: string;`)
**friendlyName for SDK type naming** — Use @friendlyName("Billing<Name>") to control the generated SDK type name and avoid collisions with Shared namespace types. (`@friendlyName("BillingParty") model BillingParty { ... }`)
**Intentional omissions documented in comments** — Fields excluded from the GOBL party spec are documented in a block comment in party.tsp with explicit rationale. Do not silently add GOBL fields. (`// Omitted fields:
// identities: maintained by apps — not needed here
// people: name is sufficient for person representation
// inboxes, emails, websites, telephones: not supported`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `party.tsp` | Defines BillingParty, BillingPartyAddresses, BillingPartyTaxIdentity — the supplier/customer party models used on invoices. | Many intentionally omitted GOBL fields are documented in a comment block (identities, people, inboxes, emails, websites, telephones, registration, logos, ext) — do not add them without design discussion. |
| `invoice.tsp` | Defines only the InvoiceNumber scalar type. Full invoice body models live in the billing/ namespace. | Do not add full invoice models here; they belong in the billing/ namespace. This file is a thin type definition only. |
| `index.tsp` | Barrel: imports invoice.tsp and party.tsp. | No namespace declaration in this index.tsp — callers import types by their fully qualified Invoices.* name. |

## Anti-Patterns

- Adding full invoice CRUD operations here — operations belong in the billing/ folder's operations.tsp
- Adding GOBL fields that are documented as intentionally omitted (identities, logos, registration, ext, etc.) without design discussion
- Defining party types without @friendlyName — generates SDK names that clash with Shared namespace types
- Using plain string instead of the InvoiceNumber scalar for invoice number fields

## Decisions

- **BillingParty is kept intentionally minimal vs full GOBL party spec** — Many GOBL fields (identity codes, people, inboxes, websites, logos) are not needed for OpenMeter's billing use case and are explicitly excluded per inline comments in party.tsp to avoid API surface bloat.

## Example: Add a new optional address type to BillingPartyAddresses

```
// party.tsp
@friendlyName("BillingPartyAddresses")
model BillingPartyAddresses {
  /**
   * Billing address.
   */
  billing_address: Shared.Address;

  /**
   * Shipping address (optional).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  shipping_address?: Shared.Address;
}
```

<!-- archie:ai-end -->
