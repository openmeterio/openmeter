# invoices

<!-- archie:ai-start -->

> TypeSpec schema-only package defining the v1 billing invoice API surface — invoice entity, line items, discounts, taxes, parties, payment terms, and document references. All files compile into api/openapi.yaml via `make gen-api`; no runtime code lives here.

## Patterns

**Lifecycle visibility on every field** — Every field must carry @visibility(Lifecycle.Read), @visibility(Lifecycle.Create), and/or @visibility(Lifecycle.Update). Read-only computed fields get only Lifecycle.Read. Missing visibility defaults to all lifecycles and leaks internal state on create/update. (`@visibility(Lifecycle.Read, Lifecycle.Create) currency: CurrencyCode;`)
**@friendlyName on every model, enum, and union** — All models, enums, and unions declare @friendlyName matching the PascalCase type name so oapi-codegen emits stable Go type names. Missing @friendlyName causes generated type name drift across regen runs. (`@friendlyName("InvoiceLineAmountDiscount") model InvoiceLineAmountDiscount { ... }`)
**Spread (...) for composition, not extends** — Shared base models (InvoiceDiscountBase, BillingParty, GenericPaymentTerms) are composed via TypeSpec spread operator (...). Direct `extends` is reserved for discriminated union variants (e.g. InvoiceCreditNoteOriginalInvoiceRef extends InvoiceGenericDocumentRef). (`model InvoiceLineUsageDiscount { ...InvoiceDiscountBase; quantity: Numeric; }`)
**Discriminated unions with envelope:none** — Polymorphic types use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) so the discriminator sits inline without an extra wrapper object. New union arms require a matching model with a typed `type` field. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union BillingDiscountReason { maximum_spend: DiscountReasonMaximumSpend, ratecard_percentage: DiscountReasonRatecardPercentage }`)
**namespace OpenMeter on all declarations** — Every .tsp file declares `namespace OpenMeter;` at the top. Omitting it causes generated OpenAPI schemas to land in the wrong component namespace, breaking Go type resolution. (`namespace OpenMeter; model Invoice { ... }`)
**main.tsp as the sole entry point** — main.tsp imports all sibling .tsp files and parent packages. New .tsp files must be listed here with `import "./newfile.tsp";` or they are silently excluded from compilation. (`import "./discounts.tsp"; import "./credits.tsp";`)
**GOBL-derived schema with explicit omission comments** — Models mirror GOBL schema but deliberately omit unsupported fields. Each omission must be documented in a comment block explaining why (e.g., `/* Omitted: exchange_rates — multi-currency not supported yet */`) so reviewers know the gap is intentional. (`/* Omitted fields: ordering, delivery — physical goods not supported */`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Package entry point — imports all sibling files and parent packages (../../productcatalog). Must be updated whenever a new .tsp file is added. | Forgetting to import a new .tsp here silently drops all its types from the generated OpenAPI. |
| `invoice.tsp` | Core Invoice and InvoiceLine models, enums (InvoiceStatus, InvoiceType, InvoiceLineStatus, InvoiceLineManagedBy), and action/detail models. Most billing API behaviour flows through these types. | InvoiceExtendedStatus is a scalar (not an enum) with @extension("x-inline", true) — do not convert to enum until statuses are fully stable. InvoiceLineReplaceUpdate uses ResourceReplaceModel with OmitProperties to exclude id and children — preserve these omissions. |
| `discounts.tsp` | Discount models split into line-level read types (InvoiceLineAmountDiscount, InvoiceLineUsageDiscount) and catalog-level input types (BillingDiscountPercentage, BillingDiscountUsage). BillingDiscountReason is a discriminated union. | BillingDiscountMetadata.correlationId is required for progressive billing coherence — do not make it non-optional without updating progressive billing logic. New discount reasons require a new union arm in BillingDiscountReason and a new DiscountReasonType enum value. |
| `party.tsp` | BillingParty and BillingInvoiceCustomerExtendedDetails models. Imports ../../customer for CustomerUsageAttribution. | @maxItems(1) on addresses — only one address per party is supported; removing this constraint requires a schema migration and Go adapter changes. |
| `pay.tsp` | PaymentTerms union (instant/dueDate) and PaymentDueDate model. GenericPaymentTerms is a generic base parameterised by PaymentTermType. | PaymentTerms is a union, not a model — adding a new payment term type requires a new union arm and a model extending GenericPaymentTerms<PaymentTermType.newType>. |
| `tax.tsp` | InvoiceLineTaxItem and InvoiceLineTaxBehavior — tax config reference on line items. Minimal subset of GOBL tax model. | A nil percent in InvoiceLineTaxItem signals tax-exempt status — this semantic must be preserved in Go adapter mapping; do not default nil to 0. |
| `docref.tsp` | InvoiceDocumentRef union for credit note back-references. Currently only creditNoteOriginalInvoice variant exists. | InvoiceDocumentRef is a union — new document reference types (proforma, debit note) must add a union arm, not extend InvoiceGenericDocumentRef directly. |
| `credits.tsp` | InvoiceLineCreditAllocation model representing credit amounts allocated to a line before taxes. | All fields are Lifecycle.Read only — credit allocations are computed by the billing engine, never set via API. |

## Anti-Patterns

- Adding runtime or business logic — this folder is schema-only; all logic belongs in openmeter/billing/
- Editing generated files (api/openapi.yaml, api/api.gen.go) instead of the .tsp source here
- Omitting @friendlyName on new models, enums, or unions — causes unstable generated Go type names
- Adding fields without @visibility decorators — fields default to all lifecycles, leaking internal state on create/update requests
- Extending InvoiceGenericDocumentRef with `extends` for new document types instead of adding a union arm to InvoiceDocumentRef

## Decisions

- **GOBL schema as the structural baseline with deliberate omissions** — GOBL provides a well-validated international invoicing schema. OpenMeter extends it for draft invoices, tiered line groups, and time-series data while explicitly documenting every omitted GOBL field so gaps are traceable and intentional rather than accidental.
- **Visibility-controlled fields instead of separate request/response types** — Using @visibility(Lifecycle.Read/Create/Update) on a single model reduces duplication and keeps the schema DRY while still producing distinct request and response shapes in the generated OpenAPI via oapi-codegen.
- **Discriminated unions with envelope:none for polymorphic types** — Avoids extra wrapping objects in the JSON payload; the discriminator sits inline matching how GOBL and most modern REST APIs represent tagged unions. Consistent with the Go-side tagged-union pattern (billing.InvoiceLine, charges.Charge) that uses private discriminators.

## Example: Adding a new invoice line discount type

```
// 1. Define the new discount model in discounts.tsp
@friendlyName("InvoiceLineNewDiscount")
model InvoiceLineNewDiscount {
  ...InvoiceDiscountBase;

  @visibility(Lifecycle.Read)
  someNewField: Numeric;
}

// 2. Add a new DiscountReasonType enum value
enum DiscountReasonType {
  maximumSpend: "maximum_spend",
  ratecardPercentage: "ratecard_percentage",
  ratecardUsage: "ratecard_usage",
  newReason: "new_reason",  // added
// ...
```

<!-- archie:ai-end -->
