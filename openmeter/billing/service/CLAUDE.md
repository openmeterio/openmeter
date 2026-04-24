# service

<!-- archie:ai-start -->

> Concrete implementation of billing.Service — the central orchestrator for profiles, customer overrides, gathering invoices, standard invoices, invoice lines, sequences, app resolution, lock acquisition, and the stateless-based invoice state machine. All DB access is delegated to billing.Adapter via entutils transactions.

## Patterns

**Config struct validation + New() constructor with RegisterLineEngine** — service.go uses Config with Validate(); New() creates the service, instantiates billinglineengine.Engine internally, and calls RegisterLineEngine so the invoice engine is always pre-registered. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; svc := &Service{...}; invoiceEngine, _ := billinglineengine.New(...); svc.RegisterLineEngine(invoiceEngine); return svc, nil }`)
**transactionForInvoiceManipulation wraps all customer-mutating operations** — All methods that create/update invoices or lines call transactionForInvoiceManipulation which first UpsertCustomerLock (outside tx) then starts a transaction and calls LockCustomerForUpdate inside it. (`return transactionForInvoiceManipulation(ctx, s, input.Customer, func(ctx context.Context) (T, error) { ... })`)
**transaction.Run / transaction.RunWithNoValue for all adapter calls** — Service methods wrap adapter access in transaction.Run or RunWithNoValue so Ent's ctx-bound transaction is propagated correctly. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.X, error) { return s.adapter.GetX(ctx, ...) })`)
**InvoiceStateMachine + sync.Pool for state machine reuse** — stdinvoicestate.go allocates InvoiceStateMachine instances from invoiceStateMachineCache (sync.Pool); external storage binds the state machine to the in-memory Invoice struct. (`sm := invoiceStateMachineCache.Get().(*InvoiceStateMachine); defer invoiceStateMachineCache.Put(sm)`)
**advancementStrategy switches between inline and queued advancement** — advanceUntilStateStable checks s.advancementStrategy: QueuedAdvancementStrategy publishes an AdvanceStandardInvoiceEvent; otherwise runs sm.AdvanceUntilStateStable inline. (`if s.advancementStrategy == billing.QueuedAdvancementStrategy { return s.publisher.Publish(ctx, billing.AdvanceStandardInvoiceEvent{...}) }`)
**featureMetersErrorWrapper converts not-found to ErrSnapshotInvalidDatabaseState** — resolveFeatureMeters wraps the returned FeatureMeters in featureMetersErrorWrapper so downstream callers receive ErrSnapshotInvalidDatabaseState instead of generic not-found for billing consistency. (`return featureMetersErrorWrapper{featureMeters}, nil`)
**resolveTaxCodes with ReadOnly flag** — taxcode.go's resolveTaxCodes uses ReadOnly=true for simulate/preview flows (GetTaxCodeByAppMapping) and ReadOnly=false for real invoice state machine transitions (GetOrCreateByAppMapping). (`taxCodes, err := s.resolveTaxCodes(ctx, resolveTaxCodesInput{Namespace: ns, Invoice: out, ReadOnly: true})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/service/service.go` | Service struct definition, Config validation, New constructor, transactionForInvoiceManipulation helper, WithAdvancementStrategy/WithLockedNamespaces options, RegisterStandardInvoiceHooks. | All invoice-mutating helpers must call transactionForInvoiceManipulation; never start a raw transaction without the customer lock sequence. |
| `openmeter/billing/service/stdinvoicestate.go` | InvoiceStateMachine with full state graph: draft → issuing → issued → payment states → paid/overdue/uncollectible/voided, plus delete and updating substates. | New states must be added to both allocateStateMachine() (Configure calls) and the StatusDetails resolver; the sync.Pool means the state machine is reused — always reset Invoice before use. |
| `openmeter/billing/service/gatheringinvoicependinglines.go` | InvoicePendingLines and prepareBillableLines — the core invoice-creation flow including progressive billing line splitting, collection cutoff resolution, and per-currency invoice creation. | resolvePendingLineCollectionCutoff has three alignment modes (bypass, subscription, anchored) — new alignment kinds must be added here. |
| `openmeter/billing/service/quantitysnapshot.go` | SnapshotLineQuantities, snapshotLineQuantitiesInParallel (semaphore-bounded), getFeatureUsage (pre-line and line-period meter queries for split lines). | Avg/Min meter aggregations are not supported for split lines — validate in getFeatureUsage.Validate(); semaphore size is maxParallelQuantitySnapshots from Config. |
| `openmeter/billing/service/invoicecalc` | Pure stateless calculation pipeline (see child summary); injected as invoicecalc.Calculator into Service; mockable via WithInvoiceCalculator. | All dependency data (FeatureMeters, TaxCodes, RatingService) must be resolved before calling Calculator.Calculate — the pipeline is stateless and cannot perform I/O. |
| `openmeter/billing/service/stdinvoiceline.go` | CreatePendingInvoiceLines: validates lines, resolves/upserts gathering invoice, normalizes and upserts lines, publishes GatheringInvoiceCreated event. | Lines with ChargeID set must have Engine explicitly set — validation enforces this; engine routing is done via lineEngines.populateGatheringLineEngine. |
| `openmeter/billing/service/profile.go` | Profile CRUD, ProvisionDefaultBillingProfile, demoteDefaultProfile, handleDefaultProfileChange (auto-pinning customers when invoicing app type changes). | Changing the default profile triggers customer re-pinning logic in handleDefaultProfileChange; all apps in a profile must have the same AppID (enforced by lo.Uniq check). |
| `openmeter/billing/service/taxcode.go` | resolveTaxCodes: collects unique Stripe tax codes from invoice lines and DefaultTaxConfig, looks them up (or creates them) via taxCodeService. | ReadOnly=true path silently skips missing codes (for simulate); ReadOnly=false creates codes (for real flows) — never use ReadOnly=false from simulate. |

## Anti-Patterns

- Calling billing.Adapter methods directly without transaction.Run/RunWithNoValue — bypasses Ent ctx-bound transaction
- Mutating customer invoices without calling transactionForInvoiceManipulation — omits advisory lock and leaves race conditions
- Calling resolveTaxCodes with ReadOnly=false from simulate or preview code paths — creates tax code rows unexpectedly
- Adding a new StandardInvoiceStatus without configuring it in allocateStateMachine() — state machine panics on unknown state
- Importing billinglineengine from outside billingservice for direct line computation — always go through billing.Service.RegisterLineEngine and the LineEngine interface

## Decisions

- **sync.Pool for InvoiceStateMachine to reduce GC pressure on high-frequency invoice advancement** — Invoice state machines are large structs with closure-bound state graphs; pooling avoids repeated allocation on the hot billing-worker path.
- **transactionForInvoiceManipulation UpsertCustomerLock outside the transaction then LockCustomerForUpdate inside** — The advisory lock row must exist before the SELECT FOR UPDATE can acquire it; the two-step pattern prevents deadlocks when concurrent invoice creation races.
- **advancementStrategy allows queued vs inline state machine advancement** — Production uses queued advancement via billing-worker Kafka events to avoid holding long transactions; tests use inline advancement for determinism.

## Example: Adding a new invoice mutation method that must use the customer lock

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
