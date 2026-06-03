# service

<!-- archie:ai-start -->

> Concrete implementation of billing.Service — the central orchestrator for profiles, customer overrides, gathering and standard invoices, invoice lines, sequences, app resolution, lock acquisition, and the stateless-backed invoice state machine. All DB access is delegated to billing.Adapter via entutils transactions.

## Patterns

**transactionForInvoiceManipulation wraps customer-mutating ops** — Methods creating/updating invoices or lines call transactionForInvoiceManipulation, which UpsertCustomerLock (outside tx) then starts a tx and calls LockCustomerForUpdate inside it. (`return transactionForInvoiceManipulation(ctx, s, input.Customer, func(ctx context.Context) (T, error) { ... })`)
**transaction.Run / RunWithNoValue for adapter calls** — Service methods wrap adapter access so Ent's ctx-bound transaction is propagated; never call the adapter bare from an exported method. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.X, error) { return s.adapter.GetX(ctx, ...) })`)
**InvoiceStateMachine pooled in sync.Pool** — stdinvoicestate.go allocates from invoiceStateMachineCache; external storage binds the state machine to the in-memory Invoice struct. (`sm := invoiceStateMachineCache.Get().(*InvoiceStateMachine); defer invoiceStateMachineCache.Put(sm)`)
**advancementStrategy switches inline vs queued** — advanceUntilStateStable publishes AdvanceStandardInvoiceEvent under QueuedAdvancementStrategy, otherwise runs sm.AdvanceUntilStateStable inline. (`if s.advancementStrategy == billing.QueuedAdvancementStrategy { return s.publisher.Publish(ctx, billing.AdvanceStandardInvoiceEvent{...}) }`)
**Config+Validate New() pre-registers the line engine** — New() builds billinglineengine.Engine internally and calls RegisterLineEngine so the invoice engine is always registered. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; svc := &Service{...}; invoiceEngine, _ := billinglineengine.New(...); svc.RegisterLineEngine(invoiceEngine); return svc, nil }`)
**resolveTaxCodes ReadOnly flag** — ReadOnly=true for simulate/preview (skips missing codes), ReadOnly=false only for real state-machine transitions (creates codes). (`taxCodes, err := s.resolveTaxCodes(ctx, resolveTaxCodesInput{Namespace: ns, Invoice: out, ReadOnly: true})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/service/service.go` | Service struct, Config validation, New constructor, transactionForInvoiceManipulation, WithAdvancementStrategy/WithLockedNamespaces options, RegisterStandardInvoiceHooks. | All invoice-mutating helpers must call transactionForInvoiceManipulation; never start a raw tx without the customer-lock sequence. |
| `openmeter/billing/service/stdinvoicestate.go` | InvoiceStateMachine full state graph (draft → issuing → issued → payment states → paid/overdue/uncollectible/voided + delete/updating). | New states must be added to allocateStateMachine() Configure calls AND the StatusDetails resolver; the sync.Pool reuses machines — reset Invoice before use. |
| `openmeter/billing/service/gatheringinvoicependinglines.go` | InvoicePendingLines / prepareBillableLines — progressive billing line splitting, collection cutoff, per-currency invoice creation. | resolvePendingLineCollectionCutoff has three alignment modes (bypass, subscription, anchored); add new kinds here. |
| `openmeter/billing/service/quantitysnapshot.go` | SnapshotLineQuantities, snapshotLineQuantitiesInParallel (semaphore-bounded), getFeatureUsage. | Avg/Min aggregations unsupported for split lines — validate in getFeatureUsage.Validate(); semaphore = maxParallelQuantitySnapshots. |
| `openmeter/billing/service/invoicecalc` | Pure stateless calculation pipeline sub-package; injected as invoicecalc.Calculator; mockable via WithInvoiceCalculator. | All dependency data (FeatureMeters, TaxCodes, RatingService) must be resolved before Calculate — the pipeline performs no I/O. |
| `openmeter/billing/service/profile.go` | Profile CRUD, ProvisionDefaultBillingProfile, demoteDefaultProfile, handleDefaultProfileChange. | Default-profile changes trigger customer re-pinning; all apps in a profile must share AppID (lo.Uniq check). |
| `openmeter/billing/service/taxcode.go` | resolveTaxCodes: gathers unique tax codes from lines + DefaultTaxConfig, looks up or creates them. | Never call with ReadOnly=false from simulate/preview paths — it creates tax-code rows. |
| `openmeter/billing/service/stdinvoiceline.go` | CreatePendingInvoiceLines: validates, resolves/upserts gathering invoice, normalizes/upserts lines, publishes GatheringInvoiceCreated. | Lines with ChargeID set must have Engine explicitly set (validated); engine routing via lineEngines.populateGatheringLineEngine. |

## Anti-Patterns

- Calling billing.Adapter methods directly without transaction.Run / RunWithNoValue — bypasses the Ent ctx-bound transaction
- Mutating customer invoices without transactionForInvoiceManipulation — omits the advisory lock and races
- Calling resolveTaxCodes with ReadOnly=false from simulate/preview paths — creates tax-code rows unexpectedly
- Adding a StandardInvoiceStatus without configuring it in allocateStateMachine() — the state machine panics on unknown state
- Importing billinglineengine from outside billingservice for direct line computation — go through RegisterLineEngine and the LineEngine interface

## Decisions

- **sync.Pool for InvoiceStateMachine to reduce GC pressure on high-frequency advancement** — State machines are large structs with closure-bound state graphs; pooling avoids repeated allocation on the hot billing-worker path.
- **Lock sequence: UpsertCustomerLock outside the tx, then LockCustomerForUpdate inside** — The advisory lock row must exist before SELECT FOR UPDATE can acquire it; the two-step pattern prevents deadlocks under concurrent invoice creation.
- **advancementStrategy allows queued vs inline state-machine advancement** — Production uses queued advancement via billing-worker Kafka events to avoid long transactions; tests use inline for determinism.

## Example: Adding an invoice mutation method that must hold the customer lock

```
func (s *Service) CancelInvoice(ctx context.Context, input billing.CancelInvoiceInput) (billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return billing.StandardInvoice{}, billing.ValidationError{Err: err}
	}
	return transactionForInvoiceManipulation(ctx, s, input.CustomerID, func(ctx context.Context) (billing.StandardInvoice, error) {
		return s.executeTriggerOnInvoice(ctx, input.Invoice, billing.TriggerVoid)
	})
}
```

<!-- archie:ai-end -->
