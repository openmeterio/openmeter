# invoices

<!-- archie:ai-start -->

> TypeSpec schema-only package defining the v1 billing invoice API surface — invoice entity, line items, discounts, taxes, parties, payment terms, document references. Compiles into api/openapi.yaml via make gen-api; no runtime code lives here.

## Patterns

**Lifecycle visibility on every field** — Every field carries @visibility(Lifecycle.Read/Create/Update). Read-only computed fields get only Lifecycle.Read. Missing visibility defaults to all lifecycles and leaks internal state on create/update. (`@visibility(Lifecycle.Read, Lifecycle.Create) currency: CurrencyCode;`)
**@friendlyName on every model, enum, and union** — All models/enums/unions declare @friendlyName matching the PascalCase type name so oapi-codegen emits stable Go type names; missing it causes type-name drift across regen runs. (`@friendlyName("InvoiceLineAmountDiscount") model InvoiceLineAmountDiscount { ... }`)
**Spread (...) for composition, not extends** — Shared base models (InvoiceDiscountBase, BillingParty, GenericPaymentTerms) compose via spread. Direct extends is reserved for discriminated union variants. (`model InvoiceLineUsageDiscount { ...InvoiceDiscountBase; quantity: Numeric; }`)
**Discriminated unions with envelope:none** — Polymorphic types use @discriminated(#{ envelope: 'none', discriminatorPropertyName: 'type' }); new arms require a matching model with a typed type field. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union BillingDiscountReason { maximum_spend: DiscountReasonMaximumSpend, ratecard_percentage: DiscountReasonRatecardPercentage }`)
**namespace OpenMeter on all declarations** — Every .tsp file declares namespace OpenMeter; at the top. Omitting it lands schemas in the wrong component namespace and breaks Go type resolution. (`namespace OpenMeter; model Invoice { ... }`)
**main.tsp as the sole entry point with documented GOBL omissions** — main.tsp imports all sibling .tsp and parent packages; new files must be added or they are silently excluded. Models mirror GOBL with deliberate omissions documented in comments. (`import "./discounts.tsp"; import "./credits.tsp";  /* Omitted: ordering, delivery — physical goods not supported */`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Package entry point — imports all sibling files and parent packages (../../productcatalog). | Forgetting to import a new .tsp here silently drops all its types from the generated OpenAPI. |
| `invoice.tsp` | Core Invoice and InvoiceLine models, enums (InvoiceStatus, InvoiceType, InvoiceLineStatus, InvoiceLineManagedBy), and action/detail models. | InvoiceExtendedStatus is a scalar (not enum) with @extension('x-inline', true) — don't convert until statuses stabilise. InvoiceLineReplaceUpdate uses ResourceReplaceModel with OmitProperties excluding id and children — preserve these omissions. |
| `discounts.tsp` | Line-level read discounts (InvoiceLineAmountDiscount, InvoiceLineUsageDiscount) and catalog-level inputs (BillingDiscountPercentage, BillingDiscountUsage); BillingDiscountReason is a discriminated union. | BillingDiscountMetadata.correlationId is required for progressive billing coherence. New discount reasons require both a BillingDiscountReason arm and a DiscountReasonType enum value. |
| `party.tsp` | BillingParty and BillingInvoiceCustomerExtendedDetails; imports ../../customer for CustomerUsageAttribution. | @maxItems(1) on addresses — only one address per party; removing it requires a schema migration and Go adapter changes. |
| `pay.tsp` | PaymentTerms union (instant/dueDate) and PaymentDueDate; GenericPaymentTerms is parameterised by PaymentTermType. | PaymentTerms is a union, not a model — a new term type needs a new arm and a model extending GenericPaymentTerms<PaymentTermType.newType>. |
| `tax.tsp` | InvoiceLineTaxItem and InvoiceLineTaxBehavior — tax config reference on line items; minimal GOBL subset. | A nil percent in InvoiceLineTaxItem signals tax-exempt; preserve this in Go adapter mapping — do not default nil to 0. |
| `docref.tsp` | InvoiceDocumentRef union for credit note back-references; currently only creditNoteOriginalInvoice. | New document reference types (proforma, debit note) must add a union arm, not extend InvoiceGenericDocumentRef directly. |
| `credits.tsp` | InvoiceLineCreditAllocation — credit amounts allocated to a line before taxes. | All fields are Lifecycle.Read only — credit allocations are computed by the billing engine, never set via API. |

## Anti-Patterns

- Adding runtime or business logic — this folder is schema-only; logic belongs in openmeter/billing/
- Editing generated files (api/openapi.yaml, api/api.gen.go) instead of the .tsp source here
- Omitting @friendlyName on new models, enums, or unions — causes unstable generated Go type names
- Adding fields without @visibility decorators — defaults to all lifecycles, leaking internal state on create/update
- Extending InvoiceGenericDocumentRef with extends for new document types instead of adding an InvoiceDocumentRef union arm

## Decisions

- **GOBL schema as the structural baseline with deliberate, documented omissions** — GOBL provides a validated international invoicing schema; OpenMeter extends it for draft invoices, tiered line groups, and time-series while documenting every omitted field so gaps are intentional.
- **Visibility-controlled fields instead of separate request/response types** — @visibility on a single model keeps the schema DRY while still emitting distinct request/response shapes via oapi-codegen.
- **Discriminated unions with envelope:none for polymorphic types** — Avoids wrapping objects in JSON; the inline discriminator matches GOBL and the Go-side tagged-union pattern (billing.InvoiceLine, charges.Charge).

## Example: Adding a new invoice line discount type

```
// 1. discounts.tsp
@friendlyName("InvoiceLineNewDiscount")
model InvoiceLineNewDiscount {
  ...InvoiceDiscountBase;
  @visibility(Lifecycle.Read)
  someNewField: Numeric;
}
// 2. add DiscountReasonType enum value: newReason: "new_reason"
```

<!-- archie:ai-end -->
