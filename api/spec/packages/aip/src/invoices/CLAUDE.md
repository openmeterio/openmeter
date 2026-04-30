# invoices

<!-- archie:ai-start -->

> TypeSpec type definitions for invoice-related models (InvoiceNumber scalar, BillingParty, tax identity, addresses) shared across billing operations in the v3 API.

## Patterns

**Namespace scoping** — All types live in `namespace Invoices;`. Files that only define types (no operations) still declare the namespace. (`namespace Invoices;
@friendlyName("BillingInvoiceNumber") scalar InvoiceNumber extends string;`)
**Scalar types for domain-constrained strings** — Use `scalar X extends string` with `@minLength`/`@maxLength`/`@example` for strongly-typed domain identifiers like InvoiceNumber. (`@example("INV-2024-01-01-01")
@minLength(1)
@maxLength(256)
scalar InvoiceNumber extends string;`)
**Visibility on every mutable field** — Every field on party models explicitly declares `@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)` or a subset — no implicit all-lifecycle fields. (`@visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
name?: string;`)
**friendlyName for SDK type naming** — Use `@friendlyName("Billing<Name>")` to control the generated SDK type name and avoid collisions with Shared namespace types. (`@friendlyName("BillingParty") model BillingParty { ... }`)

## Key Files

| File          | Role                                                                                                                        | Watch For                                                                                                                                                                                            |
| ------------- | --------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `party.tsp`   | Defines BillingParty, BillingPartyAddresses, BillingPartyTaxIdentity — the supplier/customer party models used on invoices. | Many intentionally omitted GOBL fields are documented in comments (identities, people, inboxes, emails, websites, telephones, registration, logos, ext) — do not add them without design discussion. |
| `invoice.tsp` | Defines the InvoiceNumber scalar type. Billing invoice body models live in the billing sub-folder, not here.                | This file only defines the InvoiceNumber scalar — do not add full invoice models here; they belong in the billing/ namespace.                                                                        |

## Anti-Patterns

- Adding full invoice CRUD operations here — operations belong in the billing/ folder's operations.tsp
- Adding GOBL fields that are documented as intentionally omitted (identities, logos, registration, ext, etc.)
- Defining party types without @friendlyName — generates SDK names that clash with Shared namespace

## Decisions

- **BillingParty is kept intentionally minimal vs full GOBL party spec** — Many GOBL fields (identity codes, people, inboxes, websites, logos) are not needed for OpenMeter's billing use case and are explicitly excluded per inline comments in party.tsp.

<!-- archie:ai-end -->
