# billing

<!-- archie:ai-start -->

> TypeSpec definitions for the v3 billing profile subsystem: BillingProfile, BillingWorkflow (collection/invoicing/payment/tax settings), the discriminated alignment and payment unions, TaxConfig, and BillingTotals. Compiled to the BillingProfiles* section of api/v3/openapi.yaml.

## Patterns

**Workflow settings as nested sub-models** — BillingWorkflow contains optional sub-models (collection, invoicing, payment, tax) rather than flat fields, allowing independent evolution of each settings group. (`model BillingWorkflow { collection?: BillingWorkflowCollectionSettings; invoicing?: BillingWorkflowInvoicingSettings; payment?: BillingWorkflowPaymentSettings; tax?: BillingWorkflowTaxSettings; }`)
**Discriminated union for polymorphic settings** — Payment settings and collection alignment use @discriminated(#{ envelope: "none", discriminatorPropertyName: ... }); every member model declares the discriminator field as a literal enum value. (`@discriminated(#{ discriminatorPropertyName: "collection_method", envelope: "none" }) union BillingWorkflowPaymentSettings { charge_automatically: ..., send_invoice: ... }`)
**@encode(DurationKnownEncoding.ISO8601) on duration fields** — All duration/interval fields use @encode(DurationKnownEncoding.ISO8601) and carry an @example with an ISO8601 string. (`@encode(DurationKnownEncoding.ISO8601) @example("P1D") interval?: string = "PT1H";`)
**Separate AppReferences (write) vs full App models (read)** — BillingProfileAppReferences (Read/Create only) holds AppReference objects on the write path; BillingProfileApps (Read only) holds full App objects on the read-expansion path, preventing over-posting. (`// write: apps: BillingProfileAppReferences @visibility(Read, Create)
// read expansion: BillingProfileApps @visibility(Read)`)
**@put with Shared.UpsertRequest for updates (not @patch)** — The update operation uses @put with Shared.UpsertRequest (full replacement), not @patch; diverging breaks generated SDK clients. (`@put @operationId("update-billing-profile") update(@path id: Shared.ULID, @body profile: Shared.UpsertRequest<BillingProfile>): ...`)
**Typed const for discriminated-union defaults** — DefaultBillingWorkflowCollectionAlignment is a typed const (a BillingWorkflowCollectionAlignmentSubscription value) used as the alignment default, keeping the default type-safe rather than a bare string. (`const DefaultBillingWorkflowCollectionAlignment: BillingWorkflowCollectionAlignmentSubscription = #{ type: BillingCollectionAlignmentType.Subscription };`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `profile.tsp` | Core BillingProfile and BillingWorkflow models, including all workflow sub-settings, the collection alignment union, and the payment union. | DefaultBillingWorkflowCollectionAlignment is a typed const used as the alignment default; changing alignment types requires updating both the union and the const. apps uses AppReferences on write but BillingProfileApps on read. |
| `tax.tsp` | TaxConfig, TaxBehavior, TaxConfigStripe, TaxConfigExternalInvoicing. Provider-specific tax config is additive via optional fields. | Stripe tax code must match @pattern("^txcd_\\d{8}$"); enforce @pattern on new tax-code fields. stripe/external_invoicing/tax_code_id are deprecated — prefer the tax_code reference, which takes precedence. |
| `totals.tsp` | BillingTotals read-only model with amount/taxes/charges/discounts/credits/total fields, all Shared.Numeric. | Every field is @visibility(Lifecycle.Read) — this is a computed output model, never a create/update input. |
| `operations.tsp` | BillingProfilesOperations CRUD interface using Shared.CreateRequest / UpsertRequest / CreateResponse / GetResponse / UpdateResponse / DeleteResponse. | update uses @put + Shared.UpsertRequest (full replacement), not @patch. list uses Common.PagePaginationQuery; get/update/delete add | Common.NotFound. |
| `index.tsp` | Barrel re-export importing profile.tsp, tax.tsp, operations.tsp, totals.tsp. | A new model file is invisible to the spec unless it is imported here. |

## Anti-Patterns

- Adding flat fields directly to BillingProfile instead of grouping them into a workflow settings sub-model
- Using @patch for update operations — the pattern is @put with a full UpsertRequest
- Adding mutable fields to BillingTotals — totals are always read-only computed values
- Omitting @encode(DurationKnownEncoding.ISO8601) on ISO 8601 duration string fields
- Hardcoding app references as full App models in create/update bodies — use AppReference on write paths

## Decisions

- **Payment settings modeled as a discriminated union on collection_method.** — ChargeAutomatically and SendInvoice have disjoint extra fields (due_after only applies to SendInvoice); a union avoids optional fields that are meaningless in the wrong mode.
- **BillingWorkflowCollectionAlignment defaulted via a typed const.** — A TypeSpec const lets the default be a typed discriminated-union member rather than a bare string, preserving type safety on the default value.

## Example: Add a new workflow settings group (e.g. retry settings)

```
@friendlyName("BillingWorkflowRetrySettings")
model BillingWorkflowRetrySettings {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  max_attempts?: integer = 3;
  @encode(DurationKnownEncoding.ISO8601) @example("PT5M")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  initial_delay?: string = "PT1M";
}
// then add to BillingWorkflow:  retry?: BillingWorkflowRetrySettings;
```

<!-- archie:ai-end -->
