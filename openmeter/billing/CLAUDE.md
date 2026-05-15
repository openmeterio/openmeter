# billing

<!-- archie:ai-start -->

> Core billing domain contract layer — defines the composite billing.Service interface (12 sub-interfaces: ProfileService, CustomerOverrideService, InvoiceService, GatheringInvoiceService, StandardInvoiceService, InvoiceLineService, SequenceService, LockableService, InvoiceAppService, ConfigService, LineEngineService, SplitLineGroupService), billing.Adapter interface, and all billing domain model types. No business logic lives here; this is the shared import contract for billing/service, billing/adapter, billing/httpdriver, billing/worker, app/common, and v3 API handlers.

## Patterns

**Tagged-union InvoiceLine with private discriminator** — InvoiceLine has a private field `t InvoiceLineType` set only by NewStandardInvoiceLine / NewGatheringInvoiceLine constructors. Accessed via AsStandardLine() / AsGatheringLine() / AsGenericLine() which return errors on type mismatch. Never construct with struct literals — discriminator stays zero and all accessors error. (`line := billing.NewStandardInvoiceLine(billing.StandardInvoiceLineInput{...})
std, err := line.AsStandardLine() // type-safe access`)
**Composite Service assembled from fine-grained sub-interfaces** — billing.Service embeds 12 sub-interfaces (ProfileService, CustomerOverrideService, InvoiceService, etc.). Callers depend only on the narrowest slice. When adding a new capability, add it to the appropriate sub-interface — never add methods directly to the top-level Service alias. (`// Callers that only need profile operations take ProfileService, not billing.Service
func NewProfileHandler(svc billing.ProfileService) *ProfileHandler { ... }`)
**Input.Validate() on every cross-boundary struct** — Every Input type implements Validate() error. Service methods call input.Validate() at entry. New Input types must implement Validate() and callers must invoke it before passing to service or adapter methods. (`func (u UpsertCustomerOverrideInput) Validate() error {
    if u.Namespace == "" { return fmt.Errorf("namespace is required") }
    return u.Collection.Validate()
}`)
**InvoicingApp plugin interface with read-only invoice contract** — billing.InvoicingApp (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice) is the plugin contract for Stripe/Sandbox/CustomInvoicing. Invoices passed to callbacks are read-only snapshots — mutations are silently dropped. Use UpsertResults / FinalizeStandardInvoiceResult builder and MergeIntoInvoice() to propagate external IDs back. (`result := billing.NewUpsertStandardInvoiceResult()
result.AddLineExternalID(line.ID, extID)
result.MergeIntoInvoice(invoice) // called by billing.Service, not by the app`)
**AnnotationKey/AnnotationValue constants for gathering line metadata** — consts.go and annotations.go define AnnotationKeyTaxable, AnnotationKeyReason, AnnotationValueReasonCreditPurchase, AnnotationSubscriptionSyncIgnore, AnnotationSubscriptionSyncForceContinuousLines. Always use these constants; never hardcode annotation strings inline. (`line.Annotations[billing.AnnotationKeyReason] = billing.AnnotationValueReasonCreditPurchase`)
**WorkflowConfig nested Validate() chain** — WorkflowConfig embeds CollectionConfig, InvoicingConfig, PaymentConfig, WorkflowTaxConfig — each with its own Validate(). CollectionConfig enforces AnchoredAlignmentDetail ↔ AlignmentKindAnchored consistency. WorkflowTaxConfig rejects Enforced=true when Enabled=false. Always call WorkflowConfig.Validate() before persisting. (`func (c WorkflowConfig) Validate() error {
    if err := c.Collection.Validate(); err != nil { return fmt.Errorf("collection: %w", err) }
    return c.Tax.Validate()
}`)
**UpsertResults builder for external-ID round-trip** — UpsertStandardInvoiceResult / FinalizeStandardInvoiceResult use fluent setters (SetInvoiceNumber, AddLineExternalID, SetPaymentExternalID). MergeIntoInvoice() propagates external IDs into the in-memory invoice under billing-service control. Never set fields directly on invoice inside an app callback. (`result := billing.NewUpsertStandardInvoiceResult()
result.SetInvoiceNumber("INV-001").AddLineExternalID(lineID, "ext-123")
// billing.Service calls result.MergeIntoInvoice(invoice) after callback returns`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Composite billing.Service interface definition and all 12 sub-interface declarations. The single import point for all callers of billing capabilities. | Adding implementation code here breaks the contract layer. Adding a method to a sub-interface silently breaks all existing mock implementations — regenerate wire_gen.go and update all mocks. |
| `adapter.go` | billing.Adapter interface composed of all DB access sub-interfaces plus entutils.TxCreator. Every new DB operation must appear here before the Ent adapter can implement it. | Missing TxCreator embedment breaks entutils.TransactingRepo rebinding. New adapter sub-interface methods must also be added to the concrete adapter in billing/adapter/. |
| `invoiceline.go` | InvoiceLine tagged-union type with private discriminator, constructor functions NewStandardInvoiceLine / NewGatheringInvoiceLine, and typed accessors AsStandardLine / AsGatheringLine / AsGenericLine. | Never construct InvoiceLine{} with struct literals — discriminator `t` is zero, all accessors error. Always use the constructor functions. |
| `app.go` | InvoicingApp interface, optional InvoicingAppPostAdvanceHook + InvoicingAppAsyncSyncer, UpsertResults/FinalizeStandardInvoiceResult builders, and SyncInput interface with SyncDraftStandardInvoiceInput / SyncIssuingStandardInvoiceInput implementations. | Invoice passed to app callbacks is read-only — mutations are lost. Never call billing.Service from inside app callbacks. SyncInput.Validate() and ValidateWithInvoice() must both be called. |
| `workflow.go` | WorkflowConfig and sub-configs (CollectionConfig, InvoicingConfig, PaymentConfig, WorkflowTaxConfig) with full Validate() chains. | AnchoredAlignmentDetail is only valid when Alignment == AlignmentKindAnchored. WorkflowTaxConfig.Enforced=true requires Enabled=true. DefaultWorkflowConfig in defaults.go is the canonical zero-value — do not duplicate it. |
| `errors.go` | Sentinel error types: ValidationError, NotFoundError, UpdateAfterDeleteError, AppError. HTTP layer maps these to 400/404/400/424 respectively via the billing errorEncoder. | Returning plain fmt.Errorf for domain conditions breaks HTTP status code mapping — always use the typed sentinels so the error encoder chain produces correct 4xx responses. |
| `validationissue.go` | billing.ValidationIssue (distinct from pkg/models.ValidationIssue) for invoice-level domain validation errors stored on the invoice record itself. | Do not confuse billing.ValidationIssue with models.ValidationIssue / pkg/models. They serve different purposes: billing.ValidationIssue is persisted on the invoice; models.ValidationIssue is ephemeral HTTP-layer error metadata. |
| `consts.go / annotations.go` | AnnotationKey*, AnnotationValue* constants for gathering invoice line metadata. AnnotationSubscriptionSync* constants drive subscription sync behavior. | Hardcoding annotation key/value strings instead of using these constants creates silent mismatches when constants change. |

## Anti-Patterns

- Constructing InvoiceLine{} or InvoiceGatheringLine{} with struct literals instead of NewStandardInvoiceLine/NewGatheringInvoiceLine — leaves discriminator field `t` empty, all accessor methods error at runtime
- Returning plain fmt.Errorf from service/adapter methods for domain conditions — use billing.ValidationError, billing.NotFoundError, or billing.UpdateAfterDeleteError so the HTTP error encoder maps to correct 4xx status codes
- Mutating the invoice inside InvoicingApp callbacks (ValidateStandardInvoice/UpsertStandardInvoice/FinalizeStandardInvoice) — the invoice is a read-only snapshot; mutations are silently discarded by the billing.Service
- Hardcoding annotation key/value strings instead of using the billing.AnnotationKey*/AnnotationValue* constants from consts.go and annotations.go
- Adding implementation code to service.go or adapter.go — these define only interface contracts; business logic belongs in billing/service/, persistence in billing/adapter/

## Decisions

- **Composite billing.Service assembled from 12 fine-grained sub-interfaces rather than a single monolithic interface** — Callers (worker, httpdriver, charges, subscriptionsync) depend only on the slice of billing capability they use, making mocking, testing, and future extraction of sub-domains cheaper without forcing full interface implementation.
- **InvoiceLine as a tagged union with a private discriminator rather than a Go interface or separate top-level types** — A tagged union provides exhaustive switching via Type() and type-safe accessors (AsStandardLine, AsGatheringLine) while keeping both line kinds in a single list type without reflection or interface casts at every call site.
- **UpsertResults / FinalizeStandardInvoiceResult as opaque builder structs with fluent setters rather than plain maps or direct invoice mutations** — External billing apps must not arbitrarily mutate billing domain state; the builder limits writable surface to invoice numbers, external IDs, and payment IDs, and MergeIntoInvoice applies them under billing-service control.

## Example: Implementing a new InvoicingApp integration

```
import (
    "context"
    "github.com/openmeterio/openmeter/openmeter/billing"
)

type myApp struct{}

func (a *myApp) ValidateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
    // Validate without mutating invoice — mutations are silently dropped
    return nil
}

func (a *myApp) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
    result := billing.NewUpsertStandardInvoiceResult()
    for _, line := range invoice.Lines.OrEmpty() {
// ...
```

<!-- archie:ai-end -->
