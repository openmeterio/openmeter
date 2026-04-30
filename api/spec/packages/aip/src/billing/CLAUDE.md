# billing

<!-- archie:ai-start -->

> TypeSpec definitions for the billing profile subsystem: BillingProfile, BillingWorkflow (collection, invoicing, payment, tax settings), tax config, and totals. Compiled to the BillingProfiles\* section of api/v3/openapi.yaml.

## Patterns

**Workflow settings are nested sub-models** — BillingWorkflow contains optional sub-models (collection, invoicing, payment, tax) rather than flat fields, allowing independent evolution of each settings group. (`model BillingWorkflow { collection?: BillingWorkflowCollectionSettings; invoicing?: BillingWorkflowInvoicingSettings; ... }`)
**Discriminated union for polymorphic settings** — Payment settings and collection alignment use @discriminated(#{envelope:"none", discriminatorPropertyName:...}) unions; every member model must declare the discriminator field as a literal type. (`@discriminated(#{ discriminatorPropertyName: "collection_method", envelope: "none" }) union BillingWorkflowPaymentSettings { charge_automatically: ..., send_invoice: ... }`)
**@encode(DurationKnownEncoding.ISO8601) on duration fields** — All duration/interval fields use @encode(DurationKnownEncoding.ISO8601) and carry an @example with ISO8601 string (e.g. 'P1D', 'PT1H'). (`@encode(DurationKnownEncoding.ISO8601) @example("P1D") interval?: string = "PT1H";`)
**Shared.Resource spread for profile entities** — BillingProfile spreads Shared.Resource to inherit id/created_at/updated_at. Reference types (BillingProfileReference) contain only the id field. (`model BillingProfile { ...Shared.Resource; supplier: ...; workflow: ...; }`)
**Separate References vs full models for app linkage** — BillingProfileAppReferences (used on create) holds AppReference objects; BillingProfileApps (read-only expansion) holds full App objects. This prevents over-posting. (`model BillingProfileAppReferences { tax: Apps.AppReference; invoicing: Apps.AppReference; payment: Apps.AppReference; }`)

## Key Files

| File             | Role                                                                                                                                                              | Watch For                                                                                                                                              |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `profile.tsp`    | Core BillingProfile and BillingWorkflow models. Largest file — defines all workflow sub-settings including alignment union and payment union.                     | DefaultBillingWorkflowCollectionAlignment is a const used as a default value; changing alignment types requires updating both the union and the const. |
| `tax.tsp`        | TaxConfig, TaxBehavior, TaxConfigStripe, TaxConfigExternalInvoicing. Provider-specific tax config is additive — new providers add an optional field to TaxConfig. | Stripe tax code must match pattern ^txcd\_\d{8}$; enforce @pattern on new tax code fields.                                                             |
| `totals.tsp`     | BillingTotals read-only model with amount, taxes, charges, discounts, credits, total fields all using Shared.Numeric.                                             | All fields are @visibility(Lifecycle.Read) — this is a computed output model, never a create/update input.                                             |
| `operations.tsp` | BillingProfilesOperations CRUD interface using Shared.CreateRequest, Shared.UpsertRequest, Shared.CreateResponse, Shared.GetResponse, Shared.DeleteResponse.      | Update uses @put + Shared.UpsertRequest (full replacement), not @patch. Diverging from this breaks SDK generated clients.                              |

## Anti-Patterns

- Adding flat fields directly to BillingProfile instead of grouping them into a settings sub-model (breaks the workflow settings pattern)
- Using @patch for update operations — the pattern here is @put with full UpsertRequest
- Adding mutable fields to BillingTotals — totals are always read-only computed values
- Omitting @encode(DurationKnownEncoding.ISO8601) on ISO 8601 duration string fields
- Hardcoding app references as full App models in create/update bodies — use AppReference for write paths

## Decisions

- **Payment settings modeled as discriminated union on collection_method** — ChargeAutomatically and SendInvoice have disjoint extra fields (due_after only applies to SendInvoice); a union avoids optional fields that are meaningless in the wrong mode.
- **BillingWorkflowCollectionAlignment defaulted via a const** — TypeSpec const (DefaultBillingWorkflowCollectionAlignment) lets the default value be a typed discriminated union member, not a bare string — keeping type safety on the default.

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
