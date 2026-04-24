# service

<!-- archie:ai-start -->

> Concrete implementation of subscriptionsync.Service that orchestrates the three-phase sync pipeline (load persisted state → compute target state → plan+apply reconciler diff) to keep billing artifacts (invoice lines, charges) aligned with live subscription views. Primary constraint: all mutations flow through billing.Service.WithLock to serialize per-customer operations.

## Patterns

**Config.Validate() before construction** — New(Config) calls config.Validate() first and returns an error if any required field is nil. Never construct Service directly. (`service, err := New(Config{BillingService: ..., SubscriptionService: ..., SubscriptionSyncAdapter: ..., Logger: ..., Tracer: ...}); s.NoError(err)`)
**billing.Service.WithLock wraps all mutations** — SynchronizeSubscription acquires a per-customer advisory lock via billingService.WithLock before calling buildSyncPlan + reconciler.Apply. Never mutate invoice lines or charges outside this lock. (`return s.billingService.WithLock(ctx, customerID, func(ctx context.Context) error { ... reconciler.Apply ... })`)
**Three-phase pipeline: persisted → target → plan/apply** — buildSyncPlan loads persisted state via persistedstate.NewLoader, builds target state via targetstate.NewBuilder, then calls reconciler.Plan. Apply is separate from Plan. This separation is intentional — Plan is pure in-memory. (`persisted, _ := persistedLoader.LoadForSubscription(ctx, subs); target, _ := targetBuilder.Build(ctx, ...); plan, _ := s.reconciler.Plan(ctx, PlanInput{Target: target, Persisted: persisted})`)
**tracex.Start spans on every public method** — Every exported method (SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, buildSyncPlan) opens an OTel span using tracex.StartWithNoValue or tracex.Start and wraps its body in span.Wrap. (`span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", ...); return span.Wrap(func(ctx context.Context) error { ... })`)
**FeatureFlags struct for runtime-toggleable behavior** — EnableFlatFeeInAdvanceProrating, EnableFlatFeeInArrearsProrating, EnableCreditThenInvoice are carried as a FeatureFlags struct on Service. Tests toggle them via s.Service.featureFlags = FeatureFlags{...} or s.enableProrating(). (`s.Service.featureFlags.EnableFlatFeeInAdvanceProrating = true`)
**updateSyncState after every sync branch** — All terminal code paths in SynchronizeSubscription call s.updateSyncState with the resulting MaxGenerationTimeLimit and optional PreventFurtherSyncs=true (for deleted customers). Missing this call breaks the reconciler scheduler. (`s.updateSyncState(ctx, updateSyncStateInput{SubscriptionView: subs, MaxGenerationTimeLimit: linesDiff.SubscriptionMaxGenerationTimeLimit})`)
**SuiteBase embeds billingtest.BaseSuite + SubscriptionMixin** — All test suites in this package embed SuiteBase which provides BillingService, SubscriptionService, MockStreamingConnector, ChargesService, and the setupChargesService helper. Never build test dependencies from app/common. (`type CreditsOnlySubscriptionHandlerTestSuite struct { SuiteBase }; func (s *...) SetupSuite() { s.SuiteBase.SetupSuite(); s.setupChargesService(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config, FeatureFlags, New constructor, and GetSyncStates delegation to adapter. | ChargesService is optional at construction (nil-safe); reconciler.New is called internally so reconciler.Config derives from Config — do not construct reconciler separately. |
| `sync.go` | SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, HandleSubscriptionSyncEvent, invoicePendingLines, updateSyncState — the main orchestration loop. | HasBillables() short-circuit at the top; customerDeletedAt guard for subscriptions where customer was deleted before subscription start; DryRun option must skip all adapter writes including updateSyncState. |
| `reconcile.go` | buildSyncPlan — assembles persistedstate, targetstate, and calls reconciler.Plan. Returns nil when there is nothing to do. | Returns *reconciler.Plan (pointer); callers must nil-check before calling linesDiff.IsEmpty(). |
| `handlers.go` | HandleCancelledEvent and HandleInvoiceCreation — event-driven entry points called from billing-worker Watermill handlers. | HandleCancelledEvent skips pre-sync invoice creation intentionally; HandleInvoiceCreation uses clock.Now() not the invoice timestamp to provision forward. |
| `base_test.go` | Shared test infrastructure: SuiteBase, helpers (gatheringInvoice, expectLines, recurringLineMatcher, generatePeriods, populateChildIDsFromParents). | recurringLineMatcher.ChildIDs generates UniqueIDs matching targetstate format subID/phaseKey/itemKey/v[N]/period[N] — must stay in sync with targetstate package. |
| `creditsonly_test.go` | Integration tests for credits-only settlement mode: flat fee and usage-based charge provisioning, cancellation at period boundary, mid-period shrink/prorate. | Tests use setupChargesService (not default SuiteBase setup) to inject charges.Service; absence of ChargesService means credit-only tests do not run. |
| `sync_test.go` | Integration tests for invoice-backed sync: happy path, progressive billing, cancellation, sync anchor edge cases. | Tests manipulate clock.SetTime / clock.FreezeTime; always call defer clock.ResetTime() and defer s.MockStreamingConnector.Reset() to avoid test pollution. |

## Anti-Patterns

- Calling reconciler.Apply or persistedstate.Loader outside the billing.Service.WithLock closure — concurrent invoice mutations will produce duplicate or partial lines.
- Omitting updateSyncState at the end of a new sync branch — the reconciler scheduler will re-run the subscription immediately on every tick instead of waiting for the next generation window.
- Constructing billing.Service or charges.Service directly in a test instead of using SuiteBase.setupChargesService — this bypasses the shared MockStreamingConnector and breaks streaming connector assertions.
- Using context.Background() instead of propagating the caller ctx through buildSyncPlan and reconciler calls — breaks OTel tracing and context cancellation.
- Adding business logic to HandleSubscriptionSyncEvent or HandleCancelledEvent beyond fetching the subscription view and delegating to SynchronizeSubscription — these handlers must remain thin Kafka-event adapters.

## Decisions

- **Service delegates all DB mutations to billing.Service and charges.Service interfaces rather than calling adapters directly.** — Keeps the sync service at the orchestration layer; billing.Service.WithLock, transaction management, and validation remain with the owning domain service.
- **Plan and Apply are two separate reconciler calls; buildSyncPlan returns a pure in-memory Plan with no side effects.** — Enables DryRun mode (Plan only, no Apply) and makes the diff inspectable in tests and logs before committing writes.
- **FeatureFlags is a mutable struct on Service (not injected via Wire) so integration tests can toggle individual flags per test case.** — Allows gradual feature rollout without rebuilding the service; tests use s.Service.featureFlags = FeatureFlags{} in afterTest to reset between cases.

## Example: Synchronize a subscription and immediately invoice pending lines (standard entry point from Watermill handler)

```
// sync.go
func (s *Service) SynchronizeSubscriptionAndInvoiceCustomer(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscriptionAndInvoiceCustomer", ...)
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
