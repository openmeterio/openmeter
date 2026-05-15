# billing

<!-- archie:ai-start -->

> TypeSpec definitions for the billing profile subsystem: BillingProfile, BillingWorkflow (collection/invoicing/payment/tax settings), discriminated alignment/payment unions, TaxConfig, and BillingTotals. Compiled to the BillingProfiles* section of api/v3/openapi.yaml.

## Patterns

**Workflow settings as nested sub-models** — BillingWorkflow contains optional sub-models (collection, invoicing, payment, tax) rather than flat fields, allowing independent evolution of each settings group. (`model BillingWorkflow { collection?: BillingWorkflowCollectionSettings; invoicing?: BillingWorkflowInvoicingSettings; payment?: BillingWorkflowPaymentSettings; tax?: BillingWorkflowTaxSettings; }`)
**Discriminated union for polymorphic settings** — Payment settings and collection alignment use @discriminated(#{envelope:"none", discriminatorPropertyName:...}) unions; every member model must declare the discriminator field as a literal type. (`@discriminated(#{ discriminatorPropertyName: "collection_method", envelope: "none" }) union BillingWorkflowPaymentSettings { charge_automatically: BillingWorkflowPaymentChargeAutomaticallySettings, send_invoice: BillingWorkflowPaymentSendInvoiceSettings }`)
**@encode(DurationKnownEncoding.ISO8601) on duration fields** — All duration/interval fields use @encode(DurationKnownEncoding.ISO8601) and carry an @example with ISO8601 string. (`@encode(DurationKnownEncoding.ISO8601) @example("P1D") @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) interval?: string = "PT1H";`)
**Separate AppReferences vs full App models for write vs read paths** — BillingProfileAppReferences (write: Create only) holds AppReference objects; BillingProfileApps (read-only expansion) holds full App objects, preventing over-posting. (`// Write: apps: BillingProfileAppReferences with @visibility(Read, Create)
// Read expansion: BillingProfileApps with @visibility(Read)`)
**@put with Shared.UpsertRequest for updates (not @patch)** — The update operation uses @put with Shared.UpsertRequest (full replacement), not @patch. Diverging breaks SDK generated clients. (`@put @operationId("update-billing-profile") update(@path id: Shared.ULID, @body profile: Shared.UpsertRequest<BillingProfile>): ...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `profile.tsp` | Core BillingProfile and BillingWorkflow models. Defines all workflow sub-settings including alignment union and payment union. | DefaultBillingWorkflowCollectionAlignment is a typed const used as default; changing alignment types requires updating both the union and the const. |
| `tax.tsp` | TaxConfig, TaxBehavior, TaxConfigStripe, TaxConfigExternalInvoicing. Provider-specific tax config is additive via optional fields. | Stripe tax code must match pattern ^txcd_\d{8}$; enforce @pattern on new tax code fields. Several fields are deprecated — use tax_code field instead. |
| `totals.tsp` | BillingTotals read-only model with amount/taxes/charges/discounts/credits/total fields all using Shared.Numeric. | All fields are @visibility(Lifecycle.Read) — this is a computed output model, never a create/update input. |
| `operations.tsp` | BillingProfilesOperations CRUD interface using Shared.CreateRequest, Shared.UpsertRequest, Shared.CreateResponse, Shared.GetResponse, Shared.DeleteResponse. | Update uses @put + Shared.UpsertRequest (full replacement), not @patch. |

## Anti-Patterns

- Adding flat fields directly to BillingProfile instead of grouping them into a workflow settings sub-model
- Using @patch for update operations — the pattern is @put with full UpsertRequest
- Adding mutable fields to BillingTotals — totals are always read-only computed values
- Omitting @encode(DurationKnownEncoding.ISO8601) on ISO 8601 duration string fields
- Hardcoding app references as full App models in create/update bodies — use AppReference for write paths

## Decisions

- **Payment settings modeled as discriminated union on collection_method** — ChargeAutomatically and SendInvoice have disjoint extra fields (due_after only applies to SendInvoice); a union avoids optional fields that are meaningless in the wrong mode.
- **BillingWorkflowCollectionAlignment defaulted via a typed const** — TypeSpec const (DefaultBillingWorkflowCollectionAlignment) lets the default value be a typed discriminated union member, not a bare string, keeping type safety on the default.

## Example: Add a new workflow settings group (e.g. retry settings)

```
// In profile.tsp — add sub-model:
@friendlyName("BillingWorkflowRetrySettings")
model BillingWorkflowRetrySettings {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  max_attempts?: integer = 3;

  @encode(DurationKnownEncoding.ISO8601)
  @example("PT5M")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  initial_delay?: string = "PT1M";
}
// Then add to BillingWorkflow:
//   retry?: BillingWorkflowRetrySettings;
```

<!-- archie:ai-end -->
