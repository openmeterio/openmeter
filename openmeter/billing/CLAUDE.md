# billing

<!-- archie:ai-start -->

> The largest domain in OpenMeter and the canonical billing contract layer: the package root defines the composite billing.Service (12 fine-grained sub-interfaces), billing.Adapter, the InvoiceLine tagged-union, the InvoicingApp plugin contract, and all billing domain model types. Sub-trees split the contract from its implementations — service/ orchestrates the stateless invoice state machine, adapter/ is the Ent persistence boundary, httpdriver/ exposes v1 HTTP, charges/ owns the per-type charge state machines, rating/ does pricing, worker/ runs the async event loops, and lineengine/validators/creditgrant/models supply the plugin and guard layers.

## Patterns

**Tagged-union InvoiceLine with a private discriminator** — InvoiceLine carries a private `t InvoiceLineType` set only by NewStandardInvoiceLine / NewGatheringInvoiceLine; access via AsStandardLine/AsGatheringLine/AsGenericLine which error on mismatch. The same pattern governs charges.Charge/ChargeIntent (NewCharge[T]/NewChargeIntent[T]). Struct literals leave the discriminator zero and every accessor errors at runtime. (`line := billing.NewStandardInvoiceLine(billing.StandardInvoiceLineInput{...}); std, err := line.AsStandardLine()`)
**Composite Service from fine-grained sub-interfaces; root is contract-only** — billing.Service (service.go) embeds 12 sub-interfaces (ProfileService, InvoiceService, LineEngineService, SplitLineGroupService, ...). Callers depend on the narrowest slice. Implementation lives in service/, persistence in adapter/, HTTP in httpdriver/ — never add implementation code to service.go or adapter.go. (`func NewProfileHandler(svc billing.ProfileService) *ProfileHandler { ... } // not full billing.Service`)
**Input.Validate() on every cross-boundary struct** — Every Input type implements Validate() error and service methods call it at entry; charges and rating mirror this with package-level ValidationIssue sentinels in errors.go. New Input types must implement Validate() or invalid data reaches the adapter. (`func (u UpsertCustomerOverrideInput) Validate() error { ... return u.Collection.Validate() }`)
**InvoicingApp plugin with read-only invoice + UpsertResults round-trip** — billing.InvoicingApp (ValidateStandardInvoice/UpsertStandardInvoice/FinalizeStandardInvoice/DeleteStandardInvoice) is the Stripe/Sandbox/CustomInvoicing contract. Invoices passed in are read-only snapshots; mutations are silently dropped. Propagate external IDs back via UpsertStandardInvoiceResult / FinalizeStandardInvoiceResult builders and MergeIntoInvoice (called by billing.Service, not the app). (`result := billing.NewUpsertStandardInvoiceResult(); result.AddLineExternalID(line.ID, extID)`)
**All DB access via TransactingRepo + per-customer advisory lock** — adapter/ wraps every method body in entutils.TransactingRepo; service/ wraps customer-mutating ops in transactionForInvoiceManipulation (UpsertCustomerLock outside the tx, LockCustomerForUpdate inside) and calls the adapter only through transaction.Run/RunWithNoValue. Direct a.db or billing.Adapter calls bypass the ctx tx and race. (`transaction.Run(ctx, adapter, func(ctx context.Context) (T, error) { ... })`)
**Line engines and apps register at construction, not hardcoded** — billing.Service.RegisterLineEngine wires LineEngineTypeInvoice (lineengine/) and charge engines via app/common/charges.go; InvoicingApps self-register through app.Service.RegisterMarketplaceListing. Cross-domain billing guards (validators/) register pre/post hooks into customer.Service and subscription.Service at wiring time to avoid import cycles. (`svc.RegisterLineEngine(lineengine.New(...)) // in app/common/charges.go`)
**Annotation constants for gathering-line and sync metadata** — consts.go/annotations.go define AnnotationKeyTaxable/Reason, AnnotationValueReasonCreditPurchase, AnnotationSubscriptionSyncIgnore/ForceContinuousLines etc. Always use these constants; hardcoded strings desync when constants change and break subscription sync. (`line.Annotations[billing.AnnotationKeyReason] = billing.AnnotationValueReasonCreditPurchase`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Composite billing.Service interface and all 12 sub-interface declarations — the single import point for billing capabilities. | Adding implementation code breaks the contract layer; adding a sub-interface method silently breaks all mocks and requires wire_gen.go/mock regeneration. |
| `adapter.go` | billing.Adapter interface composed of all DB-access sub-interfaces plus entutils.TxCreator. | Missing TxCreator embedment breaks TransactingRepo rebinding; new methods must also be implemented in billing/adapter/. |
| `invoiceline.go` | InvoiceLine tagged-union with private discriminator, NewStandardInvoiceLine/NewGatheringInvoiceLine constructors and typed accessors. | Never construct InvoiceLine{} via struct literal — discriminator `t` is zero and all accessors error. |
| `app.go` | InvoicingApp interface, optional InvoicingAppPostAdvanceHook/AsyncSyncer, UpsertResults/FinalizeStandardInvoiceResult builders, SyncInput implementations. | Invoice in app callbacks is read-only; never call billing.Service from inside callbacks; both SyncInput.Validate() and ValidateWithInvoice() must run. |
| `workflow.go` | WorkflowConfig and sub-configs (Collection/Invoicing/Payment/WorkflowTaxConfig) with full nested Validate() chains. | AnchoredAlignmentDetail valid only when Alignment==AlignmentKindAnchored; WorkflowTaxConfig.Enforced=true requires Enabled=true; reuse DefaultWorkflowConfig from defaults.go. |
| `errors.go` | Sentinel error types (ValidationError, NotFoundError, UpdateAfterDeleteError, AppError) mapped by the billing errorEncoder to 400/404/400/424. | Returning plain fmt.Errorf for domain conditions breaks HTTP status mapping — always use the typed sentinels. |
| `validationissue.go` | billing.ValidationIssue persisted on the invoice record (distinct from pkg/models.ValidationIssue). | Do not confuse the two: billing.ValidationIssue is persisted; models.ValidationIssue is ephemeral HTTP-layer metadata. |
| `service/service.go + service/stdinvoicestate.go` | Concrete billing.Service: orchestrates profiles, overrides, gathering/standard invoices, sequences, app resolution, and the stateless InvoiceStateMachine pooled in sync.Pool. | Adding a StandardInvoiceStatus without configuring it in allocateStateMachine() panics on unknown state; never mutate Invoice.Status directly — go through FireAndActivate. |

## Anti-Patterns

- Constructing InvoiceLine{}/charges.Charge{}/charges.ChargeIntent{} via struct literals instead of the constructors — leaves the private discriminator zero and all accessors error at runtime.
- Returning plain fmt.Errorf for domain conditions instead of billing.ValidationError/NotFoundError/UpdateAfterDeleteError — yields 500 instead of the correct 4xx.
- Mutating the invoice inside InvoicingApp callbacks — the snapshot is read-only and mutations are silently discarded; use UpsertResults + MergeIntoInvoice.
- Calling billing.Adapter or a.db directly without transaction.Run/RunWithNoValue/TransactingRepo, or mutating customer invoices without transactionForInvoiceManipulation — bypasses the ctx tx and the per-customer advisory lock.
- Adding implementation code to service.go/adapter.go, registering line engines from domain packages/cmd/* instead of app/common/charges.go, or hardcoding annotation strings.

## Decisions

- **Composite billing.Service assembled from 12 fine-grained sub-interfaces rather than one monolithic interface.** — Worker, httpdriver, charges, and subscriptionsync each depend only on the slice they use, making mocking, testing, and future sub-domain extraction cheaper.
- **InvoiceLine (and Charge/ChargeIntent) are tagged unions with private discriminators and constructor-only construction.** — Gives exhaustive, type-safe dispatch via Type()/As* accessors and makes invalid partial construction impossible, without reflection or interface casts at call sites.
- **External billing apps interact only through UpsertResults/FinalizeStandardInvoiceResult builders and a read-only invoice snapshot.** — Apps must not arbitrarily mutate billing state; the builder limits the writable surface to invoice numbers, external IDs, and payment IDs, applied under billing-service control via MergeIntoInvoice.

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
