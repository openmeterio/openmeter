# service

<!-- archie:ai-start -->

> Concrete implementation of subscriptionsync.Service that orchestrates a three-phase sync pipeline (load persisted billing state → compute target state → plan+apply reconciler diff) to keep invoice lines and charges aligned with live subscription views. All mutations are serialized per-customer via billing.Service.WithLock.

## Patterns

**Config.Validate() before construction** — New(Config) calls config.Validate() before building any internal objects and errors if BillingService, SubscriptionService, SubscriptionSyncAdapter, Logger, or Tracer is nil. ChargesService is optional (nil is valid when credits are disabled); reconciler.New is called internally from New using the same Config fields — never construct reconciler.Reconciler separately. (`service, err := New(Config{BillingService: svc, SubscriptionService: subsSvc, SubscriptionSyncAdapter: adapter, Logger: slog.Default(), Tracer: noop.NewTracerProvider().Tracer("test")})`)
**billing.Service.WithLock wraps all mutations** — SynchronizeSubscription acquires a per-customer pg_advisory_xact_lock via billingService.WithLock before calling buildSyncPlan and reconciler.Apply. No invoice-line or charge mutation may happen outside this lock closure. (`return s.billingService.WithLock(ctx, customerID, func(ctx context.Context) error { linesDiff, _ := s.buildSyncPlan(ctx, subs, asOf); return s.reconciler.Apply(ctx, reconciler.ApplyInput{Plan: linesDiff}) })`)
**Three-phase pipeline: persisted → target → plan/apply** — buildSyncPlan (reconcile.go) loads persistedstate via persistedstate.NewLoader, builds targetstate via targetstate.NewBuilder, then calls reconciler.Plan returning a pure in-memory *reconciler.Plan. Apply is a separate call enabling DryRun. Callers must nil-check the returned *Plan before IsEmpty(). (`persisted, _ := persistedstate.NewLoader(s.billingService, s.chargesService).LoadForSubscription(ctx, subs); target, _ := targetstate.NewBuilder(s.logger, s.tracer).Build(ctx, targetstate.BuildInput{...}); plan, _ := s.reconciler.Plan(ctx, reconciler.PlanInput{Target: target, Persisted: persisted})`)
**tracex.Start spans on every public method** — Every exported method (SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, invoicePendingLines, updateSyncState, buildSyncPlan) opens an OTel span via tracex.StartWithNoValue or tracex.Start and wraps its body in span.Wrap, propagating the passed ctx — never context.Background(). (`span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", trace.WithAttributes(attribute.String("subscription_id", subs.Subscription.ID))); return span.Wrap(func(ctx context.Context) error { ... })`)
**updateSyncState after every sync branch** — All terminal paths in SynchronizeSubscription — HasBillables() short-circuit, deleted-customer guard, empty plan, and successful apply — must call updateSyncState with the plan's MaxGenerationTimeLimit. Missing it makes the scheduler re-queue the subscription on every tick. (`if err := s.updateSyncState(ctx, updateSyncStateInput{SubscriptionView: subs, MaxGenerationTimeLimit: linesDiff.SubscriptionMaxGenerationTimeLimit}); err != nil { return fmt.Errorf("updating sync state: %w", err) }`)
**FeatureFlags struct for runtime-toggleable behavior** — EnableFlatFeeInAdvanceProrating, EnableFlatFeeInArrearsProrating, EnableCreditThenInvoice live in a mutable FeatureFlags struct on Service (not injected via Wire). Integration tests toggle them via s.Service.featureFlags = FeatureFlags{...} and reset them in afterTest. (`s.Service.featureFlags.EnableFlatFeeInAdvanceProrating = true // in test; s.Service.featureFlags = FeatureFlags{} // reset in afterTest`)
**Thin Kafka-event handlers delegate to sync** — HandleSubscriptionSyncEvent, HandleCancelledEvent, HandleInvoiceCreation only fetch the subscription view and delegate to SynchronizeSubscription / SynchronizeSubscriptionAndInvoiceCustomer; no business logic. HandleInvoiceCreation uses clock.Now() (not the invoice timestamp) for its provisioning horizon. (`func (s *Service) HandleCancelledEvent(ctx context.Context, ev SubscriptionCancelledEvent) error { return s.SynchronizeSubscriptionAndInvoiceCustomer(ctx, subs, clock.Now()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config with Validate(), FeatureFlags, and New constructor; reconciler.New is called internally from New. | ChargesService is optional at construction; reconciler feature flags must be set via FeatureFlags, not by configuring reconciler directly. |
| `sync.go` | Main orchestration: SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, HandleSubscriptionSyncEvent, invoicePendingLines, updateSyncState. | HasBillables() short-circuit must still call updateSyncState; customerDeletedAt guard sets PreventFurtherSyncs=true; DryRun skips all adapter writes incl. updateSyncState; billing.ErrInvoiceCreateNoLines must be swallowed in invoicePendingLines. |
| `reconcile.go` | buildSyncPlan — assembles persistedstate + targetstate and calls reconciler.Plan, returning *reconciler.Plan (nil means nothing to do). | Callers must nil-check the returned *Plan before calling IsEmpty() to avoid a nil-pointer panic. |
| `handlers.go` | HandleCancelledEvent and HandleInvoiceCreation — thin Kafka-event adapters delegating to SynchronizeSubscriptionAndInvoiceCustomer. | HandleCancelledEvent intentionally skips pre-sync invoice creation; HandleInvoiceCreation uses clock.Now(), not the invoice timestamp. |
| `base_test.go` | SuiteBase (embeds billingtest.BaseSuite + SubscriptionMixin) and shared helpers: gatheringInvoice, expectLines, setupChargesService, recurringLineMatcher. | recurringLineMatcher.ChildIDs generates UniqueIDs as subsID/phaseKey/itemKey/v[N]/period[N] — must stay in sync with the targetstate package; afterTest resets featureFlags and MockStreamingConnector. |
| `creditsonly_test.go` | Integration tests for credits-only settlement: flat-fee/usage-based charge provisioning, cancellation at period boundary, mid-period shrink/prorate. | Requires setupChargesService first; verifies idempotency by comparing charge UpdatedAt before and after a re-sync. |
| `sync_test.go` | Integration tests for invoice-backed sync: happy path, progressive billing, cancellation, continuation, billing-anchor edge cases. | Always defer clock.ResetTime() and s.MockStreamingConnector.Reset() to avoid pollution between cases. |

## Anti-Patterns

- Calling reconciler.Apply or persistedstate.Loader outside the billing.Service.WithLock closure — concurrent mutations produce duplicate or partial lines.
- Omitting updateSyncState at the end of any new sync branch — the scheduler re-queues the subscription on every tick instead of honoring MaxGenerationTimeLimit.
- Adding business logic to HandleSubscriptionSyncEvent / HandleCancelledEvent beyond fetching the view and delegating — they must stay thin Kafka adapters.
- Constructing billing.Service or charges.Service directly in a test instead of using SuiteBase / setupChargesService — bypasses the shared MockStreamingConnector and breaks streaming assertions.
- Calling billing.Adapter or charges.Adapter directly — go through billing.Service and charges.Service so WithLock, transactions, and validation stay with the owning domain.

## Decisions

- **Service delegates all DB mutations to billing.Service and charges.Service interfaces rather than adapters directly.** — Keeps the sync service at the orchestration layer; WithLock, transactions, and validation remain with the owning domain, preventing this package from accumulating persistence concerns.
- **Plan and Apply are two separate reconciler calls; buildSyncPlan returns a pure in-memory Plan with no side effects.** — Enables DryRun mode and makes the diff inspectable in logs and tests before committing writes — critical for debugging sync decisions without mutating state.
- **FeatureFlags is a mutable struct on Service rather than Wire-injected.** — Allows gradual feature rollout and per-test-case toggling; tests reset it in afterTest without reconstructing the dependency tree.

## Example: Synchronize a subscription and immediately invoice pending lines (standard Watermill-handler entry point)

```
func (s *Service) SynchronizeSubscriptionAndInvoiceCustomer(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscriptionAndInvoiceCustomer",
		trace.WithAttributes(attribute.String("subscription_id", subs.Subscription.ID)))
	return span.Wrap(func(ctx context.Context) error {
		if err := s.SynchronizeSubscription(ctx, subs, asOf); err != nil {
			return fmt.Errorf("synchronize subscription: %w", err)
		}
		customerID := customer.CustomerID{Namespace: subs.Subscription.Namespace, ID: subs.Subscription.CustomerId}
		if err := s.invoicePendingLines(ctx, customerID); err != nil {
			return fmt.Errorf("invoice pending lines (post): %w [customer_id=%s]", err, customerID.ID)
		}
		return nil
	})
}
```

<!-- archie:ai-end -->
