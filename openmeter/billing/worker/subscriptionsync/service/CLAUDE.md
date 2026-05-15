# service

<!-- archie:ai-start -->

> Concrete implementation of subscriptionsync.Service that orchestrates a three-phase sync pipeline (load persisted billing state → compute target state → plan+apply reconciler diff) to keep invoice lines and charges aligned with live subscription views. All mutations are serialized per-customer via billing.Service.WithLock.

## Patterns

**Config.Validate() before construction** — New(Config) calls config.Validate() before creating any internal objects and returns an error if BillingService, SubscriptionService, SubscriptionSyncAdapter, Logger, or Tracer is nil. ChargesService is optional (nil is valid when credits are disabled). (`service, err := New(Config{BillingService: svc, SubscriptionService: subsSvc, SubscriptionSyncAdapter: adapter, Logger: slog.Default(), Tracer: noop.NewTracerProvider().Tracer("test")})`)
**billing.Service.WithLock wraps all mutations** — SynchronizeSubscription acquires a per-customer pg_advisory_xact_lock via billingService.WithLock before calling buildSyncPlan and reconciler.Apply. No invoice line or charge mutation may happen outside this lock. (`return s.billingService.WithLock(ctx, customerID, func(ctx context.Context) error { linesDiff, _ := s.buildSyncPlan(ctx, subs, asOf, ...); return s.reconciler.Apply(ctx, reconciler.ApplyInput{Plan: linesDiff, ...}) })`)
**Three-phase pipeline: persisted → target → plan/apply** — buildSyncPlan in reconcile.go loads persistedstate via persistedstate.NewLoader, builds targetstate via targetstate.NewBuilder, then calls reconciler.Plan. Apply is a separate call. Plan is pure in-memory and enables DryRun mode. (`persisted, _ := persistedstate.NewLoader(s.billingService, s.chargesService).LoadForSubscription(ctx, subs); target, _ := targetstate.NewBuilder(s.logger, s.tracer).Build(ctx, targetstate.BuildInput{...}); plan, _ := s.reconciler.Plan(ctx, reconciler.PlanInput{Target: target, Persisted: persisted})`)
**tracex.Start spans on every public method** — Every exported method (SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, invoicePendingLines, updateSyncState, buildSyncPlan) opens an OTel span via tracex.StartWithNoValue or tracex.Start and wraps its body in span.Wrap. (`span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", trace.WithAttributes(attribute.String("subscription_id", subs.Subscription.ID))); return span.Wrap(func(ctx context.Context) error { ... })`)
**updateSyncState after every sync branch** — All terminal code paths in SynchronizeSubscription — including HasBillables() short-circuit, deleted-customer guard, empty plan, and successful apply — must call updateSyncState with the MaxGenerationTimeLimit from the plan. Missing this breaks the scheduler, causing the subscription to re-queue immediately on every tick. (`if err := s.updateSyncState(ctx, updateSyncStateInput{SubscriptionView: subs, MaxGenerationTimeLimit: linesDiff.SubscriptionMaxGenerationTimeLimit}); err != nil { return fmt.Errorf("updating sync state: %w", err) }`)
**FeatureFlags struct for runtime-toggleable behavior** — EnableFlatFeeInAdvanceProrating, EnableFlatFeeInArrearsProrating, and EnableCreditThenInvoice are carried as a FeatureFlags struct on Service. Tests toggle them via s.Service.featureFlags = FeatureFlags{...} and reset them in afterTest. (`s.Service.featureFlags.EnableFlatFeeInAdvanceProrating = true // in test; s.Service.featureFlags = FeatureFlags{} // reset in afterTest`)
**SuiteBase embeds billingtest.BaseSuite + SubscriptionMixin** — All test suites embed SuiteBase which provides BillingService, SubscriptionService, MockStreamingConnector, and the setupChargesService helper. Never build test dependencies from app/common; use setupChargesService(chargestestutils.Config{...}) to inject charges.Service when credits-only tests are needed. (`type CreditsOnlySubscriptionHandlerTestSuite struct { SuiteBase }; func (s *CreditsOnlySubscriptionHandlerTestSuite) SetupSuite() { s.SuiteBase.SetupSuite(); s.setupChargesService(chargestestutils.Config{...}) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct definition, Config with Validate(), FeatureFlags, and New constructor. reconciler.New is called internally using Config fields — do not construct reconciler.Reconciler separately outside New(). | ChargesService is optional at construction; reconciler.Config derives from the same Config fields so reconciler feature flags must be set via FeatureFlags, not by configuring reconciler directly. |
| `sync.go` | Main orchestration: SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, HandleSubscriptionSyncEvent, invoicePendingLines, updateSyncState. | HasBillables() short-circuit must still call updateSyncState; customerDeletedAt guard sets PreventFurtherSyncs=true; DryRun must skip all adapter writes including updateSyncState; billing.ErrInvoiceCreateNoLines must be swallowed in invoicePendingLines. |
| `reconcile.go` | buildSyncPlan — assembles persistedstate, targetstate, and calls reconciler.Plan. Returns *reconciler.Plan (pointer); nil means nothing to do. | Callers must nil-check the returned *Plan before calling linesDiff.IsEmpty() to avoid nil-pointer panics. |
| `handlers.go` | HandleCancelledEvent and HandleInvoiceCreation — thin Kafka-event adapters that delegate to SynchronizeSubscriptionAndInvoiceCustomer. | HandleCancelledEvent intentionally skips pre-sync invoice creation; HandleInvoiceCreation uses clock.Now() not the invoice timestamp to decide how far to provision forward. |
| `base_test.go` | SuiteBase and shared helpers: gatheringInvoice, expectLines, recurringLineMatcher, oneTimeLineMatcher, generatePeriods, populateChildIDsFromParents. | recurringLineMatcher.ChildIDs generates UniqueIDs in the format subsID/phaseKey/itemKey/v[N]/period[N] — must stay in sync with targetstate package; afterTest resets featureFlags and MockStreamingConnector. |
| `creditsonly_test.go` | Integration tests for credits-only settlement mode: flat-fee and usage-based charge provisioning, cancellation at period boundary, mid-period shrink/prorate. | Requires setupChargesService to have been called; tests verify that reconciliation is idempotent by checking charge UpdatedAt timestamps before and after a re-sync. |
| `sync_test.go` | Integration tests for invoice-backed sync: happy path, progressive billing, cancellation, continuation, billing anchor edge cases. | Always call defer clock.ResetTime() and defer s.MockStreamingConnector.Reset() to avoid test pollution between cases. |

## Anti-Patterns

- Calling reconciler.Apply or persistedstate.Loader outside the billing.Service.WithLock closure — concurrent invoice mutations produce duplicate or partial lines.
- Omitting updateSyncState at the end of any new sync branch — the reconciler scheduler re-queues the subscription immediately on every tick instead of waiting for MaxGenerationTimeLimit.
- Adding business logic to HandleSubscriptionSyncEvent or HandleCancelledEvent beyond fetching the subscription view and delegating to SynchronizeSubscription — these handlers must remain thin Kafka-event adapters.
- Constructing billing.Service or charges.Service directly in a test instead of using SuiteBase / setupChargesService — this bypasses the shared MockStreamingConnector and breaks streaming connector assertions.
- Using context.Background() instead of propagating the caller ctx through buildSyncPlan and reconciler calls — breaks OTel tracing and context cancellation.

## Decisions

- **Service delegates all DB mutations to billing.Service and charges.Service interfaces rather than calling adapters directly.** — Keeps the sync service at the orchestration layer; billing.Service.WithLock, transaction management, and validation remain with the owning domain service, preventing the sync package from accumulating persistence concerns.
- **Plan and Apply are two separate reconciler calls; buildSyncPlan returns a pure in-memory Plan with no side effects.** — Enables DryRun mode (Plan only, no Apply) and makes the diff inspectable in logs and tests before committing writes — critical for debugging sync decisions without mutating state.
- **FeatureFlags is a mutable struct on Service (not injected via Wire) so integration tests can toggle individual flags per test case.** — Allows gradual feature rollout without rebuilding the service; tests use s.Service.featureFlags = FeatureFlags{} in afterTest to reset state between cases without reconstructing the entire service dependency tree.

## Example: Synchronize a subscription and immediately invoice pending lines (standard entry point from Watermill handler)

```
// sync.go
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
