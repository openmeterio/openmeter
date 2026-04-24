# billing

<!-- archie:ai-start -->

> Core billing domain contract layer — defines billing.Service (composite interface across Profile, Invoice, Line, Sequence, App, Lock, LineEngine sub-interfaces), billing.Adapter (Ent/PostgreSQL boundary), and all domain model types (invoices, lines, profiles, customer overrides, workflow configs, split line groups, charges). It is the import contract consumed by billing/service, billing/adapter, billing/httpdriver, billing/worker, app/common, and v3 API handlers; no concrete business logic lives here.

## Patterns

**Input.Validate() on every cross-boundary struct** — Every Input type (UpsertCustomerOverrideInput, GetCustomerOverrideInput, etc.) implements a Validate() error method. Service methods call input.Validate() at entry. Adapter methods receive already-validated inputs but re-validate at the adapter boundary where the interface type is different. (`func (u UpsertCustomerOverrideInput) Validate() error { if u.Namespace == "" { return fmt.Errorf("namespace is required") } ... }`)
**Composite service interface assembled from fine-grained sub-interfaces** — billing.Service embeds ProfileService, CustomerOverrideService, InvoiceService, GatheringInvoiceService, StandardInvoiceService, InvoiceLineService, SequenceService, LockableService, InvoiceAppService, ConfigService, LineEngineService, SplitLineGroupService. Callers depend on the narrowest sub-interface needed. (`type Adapter interface { ProfileAdapter; CustomerOverrideAdapter; InvoiceLineAdapter; ... entutils.TxCreator }`)
**Tagged-union InvoiceLine via private discriminator** — InvoiceLine has a private `t InvoiceLineType` field and constructor functions NewStandardInvoiceLine / NewGatheringInvoiceLine. Access is through AsStandardLine() / AsGatheringLine() / AsGenericLine() methods that return errors on type mismatch. Never construct with struct literals. (`func (i InvoiceLine) AsStandardLine() (StandardLine, error) { if i.t != InvoiceLineTypeStandard { return StandardLine{}, fmt.Errorf("line is not a standard line") } ... }`)
**InvoicingApp interface for external billing backend plugins** — billing.InvoicingApp (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice) is the plugin contract for Stripe, Sandbox, and CustomInvoicing. Optional InvoicingAppPostAdvanceHook and InvoicingAppAsyncSyncer extend it. Use GetApp(app.App) to type-assert at runtime. (`func GetApp(app app.App) (InvoicingApp, error) { customerApp, ok := app.(InvoicingApp); if !ok { return nil, AppError{...} } return customerApp, nil }`)
**UpsertResults builder pattern for external-ID round-trip** — UpsertStandardInvoiceResult / FinalizeStandardInvoiceResult use fluent setters (SetInvoiceNumber, AddLineExternalID, SetPaymentExternalID). MergeIntoInvoice() propagates external IDs back into the in-memory invoice after the external billing app responds. Never set fields directly on the invoice inside an app callback. (`result.AddLineExternalID(line.ID, extID); result.MergeIntoInvoice(invoice)`)
**AnnotationKey/AnnotationValue constants for gathering line metadata** — consts.go and annotations.go define AnnotationKeyTaxable, AnnotationKeyReason, AnnotationValueReasonCreditPurchase, AnnotationSubscriptionSyncIgnore, AnnotationSubscriptionSyncForceContinuousLines. Always use these constants; never hardcode annotation strings inline. (`line.Annotations[billing.AnnotationKeyReason] = billing.AnnotationValueReasonCreditPurchase`)
**WorkflowConfig with nested Validate() chain** — WorkflowConfig embeds CollectionConfig, InvoicingConfig, PaymentConfig, WorkflowTaxConfig each with their own Validate(). CollectionConfig enforces AnchoredAlignmentDetail ↔ AlignmentKindAnchored consistency. WorkflowTaxConfig rejects Enforced=true when Enabled=false. Always call WorkflowConfig.Validate() before persisting. (`func (c WorkflowConfig) Validate() error { if err := c.Collection.Validate(); err != nil { return ... } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/service.go` | Composite billing.Service interface definition and all sub-interface declarations. The single import point for all callers of billing capabilities. | Do not add implementation code here; only interface signatures. Adding a method to a sub-interface silently breaks all existing mock implementations. |
| `openmeter/billing/adapter.go` | billing.Adapter interface composed of all DB access sub-interfaces (ProfileAdapter, CustomerOverrideAdapter, InvoiceLineAdapter, etc.) plus entutils.TxCreator. | Every new DB operation must be added here before the Ent adapter can implement it. Missing TxCreator embedment breaks entutils.TransactingRepo rebinding. |
| `openmeter/billing/invoiceline.go` | InvoiceLine tagged-union type with private discriminator, constructor functions, and typed accessor methods (AsStandardLine, AsGatheringLine, AsGenericLine). | Never construct InvoiceLine{} with struct literals — the discriminator `t` will be zero and all accessor methods will error. Always use NewStandardInvoiceLine / NewGatheringInvoiceLine. |
| `openmeter/billing/app.go` | InvoicingApp interface, optional InvoicingAppPostAdvanceHook + InvoicingAppAsyncSyncer interfaces, UpsertResults / FinalizeStandardInvoiceResult builder types, SyncInput interface and its two implementations. | App callbacks receive a read-only invoice snapshot — mutations are lost. Do not call billing.Service methods from inside app callbacks. |
| `openmeter/billing/workflow.go` | WorkflowConfig and its sub-configs (CollectionConfig, InvoicingConfig, PaymentConfig, WorkflowTaxConfig) with full Validate() chains. | AnchoredAlignmentDetail is only valid when Alignment == AlignmentKindAnchored. WorkflowTaxConfig.Enforced=true requires Enabled=true. |
| `openmeter/billing/customeroverride.go` | CustomerOverride domain model, all override input/output types, ListCustomerOverridesInput with ordering constants, BulkAssignCustomersToProfileInput. | CustomersWithoutPinnedProfile and BillingProfiles are mutually exclusive in ListCustomerOverridesInput.Validate(). |
| `openmeter/billing/validationissue.go` | billing.ValidationIssue (distinct from models.ValidationIssue) for invoice-level domain validation errors stored on the invoice record itself. | Do not confuse billing.ValidationIssue with models.ValidationIssue / pkg/models. They serve different purposes and are used in different layers. |
| `openmeter/billing/errors.go` | Sentinel error types: ValidationError, NotFoundError, UpdateAfterDeleteError, AppError. HTTP layer maps these to 400/404/400/424 respectively. | Returning a plain fmt.Errorf instead of one of these typed errors breaks HTTP status code mapping in the billing errorEncoder chain. |

## Anti-Patterns

- Constructing InvoiceLine{} with struct literals instead of NewStandardInvoiceLine/NewGatheringInvoiceLine — leaves discriminator field `t` empty, all accessors error
- Returning plain fmt.Errorf from service/adapter methods for domain conditions — use billing.ValidationError, billing.NotFoundError, or billing.UpdateAfterDeleteError so the HTTP error encoder maps to correct status codes
- Mutating the invoice inside InvoicingApp callbacks — the invoice passed to ValidateStandardInvoice/UpsertStandardInvoice/FinalizeStandardInvoice is a read-only snapshot; mutations are silently discarded
- Hardcoding annotation key/value strings instead of using the billing.AnnotationKey*/AnnotationValue* constants from consts.go and annotations.go
- Adding implementation code to service.go or adapter.go — these files define only interface contracts; business logic belongs in billing/service/, persistence in billing/adapter/

## Decisions

- **Composite billing.Service assembled from fine-grained sub-interfaces rather than a single monolithic interface** — Callers (worker, httpdriver, charges, subscriptionsync) depend only on the slice of billing capability they actually use, making mocking, testing, and future extraction of sub-domains cheaper.
- **InvoiceLine as a tagged union with a private discriminator field rather than a Go interface or separate top-level types** — A tagged union provides exhaustive switching via Type() and type-safe accessors (AsStandardLine, AsGatheringLine) while keeping the two line kinds in a single list type without reflection; a Go interface would require runtime type assertions everywhere.
- **UpsertResults / FinalizeStandardInvoiceResult as opaque builder structs with fluent setters rather than plain maps or direct invoice mutations** — External billing apps (Stripe, CustomInvoicing) must not be able to arbitrarily mutate billing domain state; the builder limits writable surface to invoice numbers, external IDs, and payment IDs, and MergeIntoInvoice applies them under billing-service control.

## Example: Implementing a new InvoicingApp integration

```
import (
    "context"
    "github.com/openmeterio/openmeter/openmeter/billing"
)

type myApp struct{}

func (a *myApp) ValidateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
    // validate without mutating invoice
    return nil
}

func (a *myApp) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
    result := billing.NewUpsertStandardInvoiceResult()
    // sync to external system, collect external IDs
// ...
```

<!-- archie:ai-end -->
