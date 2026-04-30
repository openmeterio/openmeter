# invoices

<!-- archie:ai-start -->

> TypeSpec models for the v1 billing invoice API surface — defines the invoice entity, line items, discounts, taxes, parties, payment terms, and document references. All files compile into api/openapi.yaml via `make gen-api`; no runtime code lives here.

## Patterns

**Lifecycle visibility on every field** — Each field must carry @visibility(Lifecycle.Read), @visibility(Lifecycle.Create), and/or @visibility(Lifecycle.Update) to control which operations expose it. Read-only computed fields get only Lifecycle.Read. (`@visibility(Lifecycle.Read, Lifecycle.Create) currency: CurrencyCode;`)
**@friendlyName on every model and union** — All models, enums, and unions declare @friendlyName matching the PascalCase type name so oapi-codegen emits stable Go type names. Missing @friendlyName causes generated type name drift. (`@friendlyName("InvoiceLineAmountDiscount") model InvoiceLineAmountDiscount { ... }`)
**Spread (…) for composition, not inheritance** — TypeSpec spread operator (...) is used to compose shared base models (ResourceTimestamps, InvoiceDiscountBase, BillingParty). Direct extends is reserved for discriminated union variants. (`model InvoiceLineUsageDiscount { ...InvoiceDiscountBase; quantity: Numeric; }`)
**Discriminated unions with envelope:none** — Polymorphic types use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) so the discriminator sits inline without an extra wrapper object. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union BillingDiscountReason { maximum_spend: DiscountReasonMaximumSpend, ... }`)
**Namespace OpenMeter on all declarations** — Every .tsp file in this folder is in namespace OpenMeter. Omitting the namespace causes the generated OpenAPI schema to land in the wrong component namespace. (`namespace OpenMeter; model Invoice { ... }`)
**main.tsp as the sole entry point** — main.tsp imports all sibling .tsp files and parent packages. New .tsp files must be added to main.tsp imports or they will not be compiled. (`import "./discounts.tsp";`)
**GOBL-derived schema with explicit omission comments** — Models mirror GOBL schema but deliberately omit unsupported fields. Each omission must be documented in a comment block explaining why (e.g. '/_ Omitted: multi-currency exchange_rates — not supported yet _/') so reviewers know the gap is intentional. (`/* Omitted fields: ordering, delivery — physical goods not supported */`)

## Key Files

| File            | Role                                                                                                                                                                                                                                    | Watch For                                                                                                                                                     |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `main.tsp`      | Package entry point — imports all sibling files and parent packages (../../productcatalog). Must be updated whenever a new .tsp file is added.                                                                                          | Forgetting to import a new .tsp here silently drops all its types from the generated OpenAPI.                                                                 |
| `invoice.tsp`   | Core Invoice and InvoiceLine models, enums (InvoiceStatus, InvoiceType, InvoiceLineStatus, InvoiceLineManagedBy), and action/detail models. Most billing API behaviour flows through these types.                                       | InvoiceExtendedStatus is a scalar (not an enum) with @extension("x-inline", true) — do not convert to enum until status values are stable.                    |
| `discounts.tsp` | Discount models split into line-level read types (InvoiceLineAmountDiscount, InvoiceLineUsageDiscount) and catalog-level input types (BillingDiscountPercentage, BillingDiscountUsage). BillingDiscountReason is a discriminated union. | BillingDiscountMetadata.correlationId is required for progressive billing coherence — do not make it non-optional without updating progressive billing logic. |
| `party.tsp`     | BillingParty and BillingInvoiceCustomerExtendedDetails models. Imports ../../customer for CustomerUsageAttribution.                                                                                                                     | @maxItems(1) on addresses — only one address per party is supported; do not remove this constraint without a schema migration.                                |
| `pay.tsp`       | PaymentTerms union (instant/due_date) and PaymentDueDate model. GenericPaymentTerms is a generic base parameterised by PaymentTermType.                                                                                                 | PaymentTerms is a union, not a model — adding a new payment term type requires a new union arm and a model extending GenericPaymentTerms.                     |
| `tax.tsp`       | InvoiceLineTaxItem and InvoiceLineTaxBehavior — tax config reference on line items. Minimal subset of GOBL tax model.                                                                                                                   | A nil percent in InvoiceLineTaxItem signals tax-exempt status — this semantic must be preserved in Go adapter mapping.                                        |
| `docref.tsp`    | InvoiceDocumentRef union for credit note back-references. Currently only creditNoteOriginalInvoice variant exists.                                                                                                                      | InvoiceDocumentRef is a union — new document reference types (proforma, debit note) must add a union arm, not extend InvoiceGenericDocumentRef directly.      |

## Anti-Patterns

- Adding runtime or business logic — this folder is schema-only; all logic belongs in openmeter/billing/
- Editing generated files (api/openapi.yaml, api/api.gen.go) instead of the .tsp source here
- Omitting @friendlyName on new models — causes unstable generated Go type names
- Adding fields without @visibility decorators — fields default to all lifecycles which exposes internal state on create/update
- Extending InvoiceGenericDocumentRef with extends instead of adding a union arm to InvoiceDocumentRef

## Decisions

- **GOBL schema as the structural baseline with deliberate omissions** — GOBL provides a well-validated international invoicing schema. OpenMeter extends it for draft invoices, tiered line groups, and time-series data while explicitly documenting every omitted GOBL field so gaps are traceable and intentional.
- **Visibility-controlled fields instead of separate request/response types** — Using @visibility(Lifecycle.Read/Create/Update) on a single model reduces duplication and keeps the schema DRY while still producing distinct request and response shapes in the generated OpenAPI via oapi-codegen.
- **Discriminated unions with envelope:none for polymorphic reason/term types** — Avoids extra wrapping objects in the JSON payload; the discriminator sits inline matching how GOBL and most modern REST APIs represent tagged unions.

## Example: Adding a new invoice line discount type

```
// 1. Define the new discount model in discounts.tsp
@friendlyName("InvoiceLineNewDiscount")
model InvoiceLineNewDiscount {
  ...InvoiceDiscountBase;

  @visibility(Lifecycle.Read)
  someNewField: Numeric;
}

// 2. Add it to InvoiceLineDiscounts
@friendlyName("InvoiceLineDiscounts")
model InvoiceLineDiscounts {
  amount?: InvoiceLineAmountDiscount[];
  usage?: InvoiceLineUsageDiscount[];
  newType?: InvoiceLineNewDiscount[];  // added arm
// ...
```

<!-- archie:ai-end -->
