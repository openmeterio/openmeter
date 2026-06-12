# service

<!-- archie:ai-start -->

> Implements subscriptionsync.Service: the orchestration layer that synchronizes a subscription's billable state into billing artifacts (invoice lines + charges). It drives the load-persisted / build-target / repair / Plan / Apply pipeline under a per-customer billing lock and persists sync state. Primary constraint: every public sync entrypoint runs inside billingService.WithLock, reconciler.Plan stays pure, and reconciler.Apply is the only writer.

## Patterns

**Plan/Apply split via buildSyncPlan then reconciler.Apply** — synchronizeSubscription always calls buildSyncPlan (pure: loads persistedstate, builds targetstate, repairs charge refs, calls s.reconciler.Plan) and only writes through s.reconciler.Apply. No mutation happens during planning. (`linesDiff, err := s.buildSyncPlan(ctx, subs, subsView, asOf, customerDeletedAt, currency, options.DryRun); ... s.reconciler.Apply(ctx, reconciler.ApplyInput{DryRun: options.DryRun, Customer: customerID, Currency: currency, Plan: linesDiff})`)
**All sync work runs under withBillingLock** — The Plan->Apply->updateSyncState body of synchronizeSubscription executes inside withBillingLock(ctx, s, customerID, fn), which delegates to s.billingService.WithLock keyed on the customer. Never write billing/charge state outside this lock. (`return withBillingLock(ctx, s, customer.CustomerID{Namespace: subs.Namespace, ID: subs.CustomerId}, func(ctx context.Context) (*synchronizeSubscriptionResult, error) { ... })`)
**subscriptionReferenceOrView indirection** — Public entrypoints accept either a NamespacedID or a SubscriptionView; newSubscriptionReferenceOrView normalizes both (plus subscription.Subscription) into one ref type. Use AsSubscriptionView/AsNamespacedID/GetID rather than branching on raw types. (`func (s *Service) SyncByView(ctx, view, asOf, opts...) error { _, err := s.synchronizeSubscription(ctx, newSubscriptionReferenceOrView(view), asOf, opts...); return err }`)
**Config.Validate gates construction and delegates to reconciler.New** — New(Config) calls config.Validate() (BillingService, SubscriptionService, SubscriptionSyncAdapter, Logger, Tracer, FeatureGate all required) then builds the reconciler from FeatureFlags. ChargesService is optional (required only for credit-only / charge-based sync). (`if err := config.Validate(); err != nil { return nil, err }; reconcilerSvc, err := reconciler.New(reconciler.Config{BillingService: config.BillingService, ChargesService: config.ChargesService, EnableCreditThenInvoice: config.FeatureFlags.EnableCreditThenInvoice, CreditsFlag: config.FeatureFlags.CreditsFlag})`)
**tracex span wrapping on every operation** — Each meaningful method opens tracex.Start / tracex.StartWithNoValue and returns span.Wrap(func(ctx)...). buildSyncPlan, synchronizeSubscription, invoicePendingLines, updateSyncState all follow this; errors propagate up through Wrap rather than panicking. (`span := tracex.Start[*reconciler.Plan](ctx, s.tracer, "billing.worker.subscription.sync.buildSyncPlan"); return span.Wrap(func(ctx context.Context) (*reconciler.Plan, error) { ... })`)
**DryRun short-circuits before every write and state upsert** — options.DryRun (from SynchronizeSubscriptionOptions) is threaded into buildSyncPlan, reconciler.Apply, and returns before updateSyncState. Dry runs never call UpsertSyncState or persist charge item-id repairs. (`if options.DryRun { return res, nil }  // returned before updateSyncState`)
**Sync-state persistence via updateSyncState** — After applying, updateSyncState upserts subscriptionsync sync state through the adapter: HasBillables drives scheduling, MaxGenerationTimeLimit becomes NextSyncAfter, and PreventFurtherSyncs forces HasBillables=false (used when the customer was deleted before subscription start). (`s.subscriptionSyncAdapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{SubscriptionID: in.SubscriptionID, HasBillables: true, NextSyncAfter: lo.ToPtr(nextSyncAfter), SyncedAt: clock.Now().UTC()})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config/FeatureFlags, New(), and the public Sync* entrypoints (SyncByView, SyncByID, SyncByViewAndInvoiceCustomer, SyncByIDAndInvoiceCustomer, GetSyncStates). Asserts var _ subscriptionsync.Service = (*Service)(nil). | ChargesService is nil-tolerant; repairChargeSubscriptionReferences and credit paths guard on it. Adding a required dep means updating Config.Validate. |
| `sync.go` | Core synchronizeSubscription / synchronizeSubscriptionAndInvoiceCustomer pipeline, withBillingLock, getSubscription (List with IncludeDeleted), updateSyncState, invoicePendingLines, and HandleSubscriptionSyncEvent. | Deleted subscriptions => subsView=nil (Plan must tolerate nil view). invoicePendingLines swallows billing.ErrInvoiceCreateNoLines; do not treat 'no lines' as failure. |
| `reconcile.go` | buildSyncPlan: wires persistedstate.NewLoader, targetstate.NewBuilder, repairChargeSubscriptionReferences, then reconciler.Plan. The single bridge from sub view + persisted state to a reconciler.Plan. | Order matters: repair runs AFTER target build and BEFORE Plan; the repaired persisted state is what is fed into Plan. |
| `repair.go` | repairChargeSubscriptionReferences: realigns persisted charges' subscription_item_id when subscription edits recreated the item row under the same logical (sub/phase/item/version) identity. Helpers WithSubscriptionItemID / UpdateSubscriptionItemID and persistedItemFromCharge. | Deliberately narrow: only subscription_item_id is mutated; subscription_id/phase_id mismatches are hard integrity errors. dryRun uses in-memory WithSubscriptionItemID; otherwise UpdateSubscriptionItemID persists. Carries a TODO to make item id immutable again. |
| `handlers.go` | Event handlers: HandleCancelledEvent, HandleInvoiceCreation (backfills the gathering invoice for affected subscriptions), HandleDeletedEvent. Thin wrappers over synchronizeSubscription*. | HandleInvoiceCreation uses clock.Now() (not the invoice time) as the sync reference to provision more lines if delayed; HandleCancelledEvent returns an error if event.Spec.ActiveTo is nil after a best-effort sync. |
| `ref.go` | subscriptionReferenceOrView value type + newSubscriptionReferenceOrView generic constructor (NamespacedID | SubscriptionView | Subscription) and accessors (Type, AsNamespacedID, AsSubscriptionView, GetID, Validate). | GetID/AsX return zero values or errors for the wrong reference type; always check Type() or handle the error rather than assuming a view is present. |
| `base_test.go` | SuiteBase test harness embedding billingtest.BaseSuite + SubscriptionMixin; builds the real adapter + Service, setupChargesService wires chargestestutils, and provides expectLines/assertCharges matchers. | afterTest resets featureFlags and MockStreamingConnector; enableProrating flips featureFlags directly. Tests drive behavior through SyncByView/SyncByViewAndInvoiceCustomer, not lower-level adapters. |
| `sync_credittheninvoice_test.go` | End-to-end CreditThenInvoice scenarios wiring a full ledger stack (ledgertestutils, ledgercollector, ledgerchargeadapter handlers) with FeatureFlags{EnableCreditThenInvoice:true, CreditsFlag:"billing_credits"}. | Requires EnsureBusinessAccounts + CreateCustomerAccounts in BeforeTest; credit-then-invoice charges route through usagebased charges, not plain invoice lines. |

## Anti-Patterns

- Writing billing or charge state outside withBillingLock / billingService.WithLock — breaks per-customer serialization across replicas.
- Mutating persisted billing artifacts inside buildSyncPlan or reconciler.Plan; planning must stay pure and only reconciler.Apply may write.
- Skipping the options.DryRun guards and calling UpsertSyncState / persisting charge item-id repairs during a dry run.
- Repairing more than subscription_item_id in repairChargeSubscriptionReferences, or swallowing a subscription_id/phase_id mismatch instead of returning the integrity error.
- Introducing context.Background()/time.Now() ad hoc instead of propagating ctx and using clock.Now(); panicking on a bad subscription/period instead of returning an error through span.Wrap.

## Decisions

- **Split planning (buildSyncPlan -> reconciler.Plan) from writing (reconciler.Apply), with persistedstate/targetstate as inputs.** — Lets sync be dry-runnable and keeps the diff logic deterministic and side-effect-free; the reconciler is the single mutation point so atomicity and DryRun are enforced in one place.
- **Identify billable items by logical path (sub/phase/item/version/period), not by subscription_items.id, and add a narrow repair step to realign concrete item ids.** — Subscription edits can soft-delete and recreate an item row for the same logical item; logical identity keeps reconciliation as update/shrink rather than delete+recreate, while repair keeps charge rows pointing at the live item id.
- **ChargesService is optional in Config; reconciler and repair guard on nil.** — Charge-based / credit-only sync (and the credits feature) can be disabled, so the same Service supports invoice-line-only deployments without charges wiring.

## Example: buildSyncPlan: load persisted, build target, repair, then produce a pure reconciler.Plan

```
func (s *Service) buildSyncPlan(ctx context.Context, subs subscription.Subscription, subsView *subscription.SubscriptionView, asOf time.Time, customerDeletedAt *time.Time, currency currencyx.Calculator, dryRun bool) (*reconciler.Plan, error) {
  span := tracex.Start[*reconciler.Plan](ctx, s.tracer, "billing.worker.subscription.sync.buildSyncPlan")
  return span.Wrap(func(ctx context.Context) (*reconciler.Plan, error) {
    persisted, err := persistedstate.NewLoader(s.billingService, s.chargesService).LoadForSubscription(ctx, subs)
    if err != nil { return nil, err }
    target, err := targetstate.NewBuilder(s.logger, s.tracer).Build(ctx, targetstate.BuildInput{AsOf: asOf, CustomerDeletedAt: customerDeletedAt, SubscriptionView: subsView, Persisted: persisted})
    if err != nil { return nil, err }
    persisted, err = s.repairChargeSubscriptionReferences(ctx, persisted, target, dryRun)
    if err != nil { return nil, err }
    return s.reconciler.Plan(ctx, reconciler.PlanInput{SubscriptionSettlementMode: subs.SettlementMode, Currency: currency, Target: target, Persisted: persisted})
  })
}
```

<!-- archie:ai-end -->
