# service

<!-- archie:ai-start -->

> Implements the usagebased.Service interface — charge creation, state-machine-driven advancement, patch triggering, real-time usage queries, and billing.LineEngine integration. It composes two sub-packages (rating for ClickHouse queries, run for realization mechanics) and dispatches each charge to the correct settlement-mode state machine (CreditsOnly or CreditThenInvoice).

## Patterns

**Config-struct constructor with Validate()** — Every exported constructor (New, newStateMachineBase, NewCreditsOnlyStateMachine, NewCreditThenInvoiceStateMachine) accepts a typed Config/StateMachineConfig and calls config.Validate() via errors.Join before allocating. Missing deps produce a joined error list. (`func New(config Config) (usagebased.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**withLockedCharge wraps every mutating operation** — AdvanceCharge and TriggerPatch always call withLockedCharge which opens a transaction via transaction.Run, acquires a pg advisory lock via s.locker.LockForTX using charges.NewLockKeyForCharge, then fetches the charge with ExpandRealizations before delegating to the callback. Never mutate charge state outside this wrapper. (`return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) { ... })`)
**Settlement-mode dispatch in newStateMachine** — triggers.go newStateMachine switches on charge.Intent.SettlementMode: CreditOnlySettlementMode → CreditsOnlyStateMachine, CreditThenInvoiceSettlementMode → CreditThenInvoiceStateMachine; unsupported modes return models.NewGenericNotImplementedError. (`switch config.Charge.Intent.SettlementMode { case productcatalog.CreditOnlySettlementMode: ... case productcatalog.CreditThenInvoiceSettlementMode: ... default: return nil, models.NewGenericNotImplementedError(...) }`)
**FireAndActivate + AdvanceUntilStateStable bookend in LineEngine hooks** — LineEngine hooks (OnStandardInvoiceCreated, OnCollectionCompleted, OnInvoiceIssued) always call AdvanceUntilStateStable before firing the invoice trigger and again after so all auto-transitions complete before reading the run result. (`stateMachine.AdvanceUntilStateStable(ctx); stateMachine.FireAndActivate(ctx, trigger, input); stateMachine.AdvanceUntilStateStable(ctx)`)
**GetLineEngine() is the sole LineEngine construction path** — service.go exposes GetLineEngine() returning &LineEngine{service: s}. Callers (billing.Service) register this engine via RegisterLineEngine(billing.LineEngineTypeChargeUsageBased, ...). LineEngine must not be constructed directly — it requires access to rater, runs, adapter, and locker all owned by service. (`func (s *service) GetLineEngine() billing.LineEngine { return &LineEngine{service: s} }`)
**Parallel rating with semaphore in expandChargesUsage** — get.go uses semaphore.NewWeighted(defaultMaxParallelRatingsPerRequest=5) + sync.WaitGroup for concurrent ClickHouse rating across multiple charges, with panic recovery per goroutine and errors.Join on all channel errors. (`sem := semaphore.NewWeighted(int64(defaultMaxParallelRatingsPerRequest)); sem.Acquire(ctx,1); ... defer sem.Release(1)`)
**populateUsageBasedStandardLineFromRun — totals must match after projection** — linemapper.go populateUsageBasedStandardLineFromRun projects run DetailedLines onto stdLine and verifies stdLine.Totals equals run.Totals after RoundToPrecision; mismatch is a fatal error. Always call runs.MapToBillingMeteredQuantity and mutator.ApplyUsageDiscount before computing quantities. (`if !stdLine.Totals.Equal(expectedTotals) { return fmt.Errorf("projected line totals do not match run totals ...") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Constructor and struct definition. Composes usagebasedrating.Service and usagebasedrun.Service (constructed internally). Exposes GetLineEngine(). All Config fields are required. | Do not add business logic here. All Config fields must be non-nil or Validate() returns a joined error — add new fields with a matching nil check in Validate(). |
| `triggers.go` | AdvanceCharge and TriggerPatch entry points. withLockedCharge helper (transaction + advisory lock + re-fetch). newStateMachine settlement-mode dispatch. getStateMachineConfigForPatch re-fetches customerOverride + featureMeter. | Every mutating path must go through withLockedCharge. getStateMachineConfigForPatch re-fetches customer and feature on every call — avoid calling it in hot loops. |
| `statemachine.go` | Base stateMachine struct embedding chargestatemachine.Machine. Shared guards (IsInsideServicePeriod, IsAfterServicePeriod, IsAfterCollectionPeriod). AdvanceAfter setters. ensureDetailedLinesLoadedForRating. | IsAfterCollectionPeriod silently returns false on error (logging only) — do not rely on it for critical decisions. ensureDetailedLinesLoadedForRating mutates s.Charge in place. |
| `creditheninvoice.go` | CreditThenInvoiceStateMachine: full partial+final invoice state graph, StartPartialInvoiceRun, StartFinalInvoiceRun, SnapshotInvoiceUsage, FinalizeInvoiceRun, Extend/Shrink/Delete patch handlers. | SnapshotInvoiceUsage reads CurrentRealizationRunID — fail fast if nil. FinalizeInvoiceRun clears CurrentRealizationRunID and AdvanceAfter after BookAccruedInvoiceUsage. Extend/Shrink in issuing/completed states return GenericPreConditionFailedError. |
| `creditsonly.go` | CreditsOnlyStateMachine: simpler graph with no invoice steps, final realization only. DeleteCharge corrects credits on CreditRefundPolicyCorrect. Uses CreditAllocationExact. | storedAtLT is clock.Now() minus InternalCollectionPeriod. CreditAllocationExact (not CreditAllocationAvailable) for credits-only charges. |
| `lineengine.go` | billing.LineEngine implementation: SplitGatheringLine, BuildStandardInvoiceLines, OnStandardInvoiceCreated, OnCollectionCompleted, OnInvoiceIssued, OnPaymentAuthorized, OnPaymentSettled, OnMutableStandardLinesDeleted. | SplitGatheringLine clears ChildUniqueReferenceID on both halves. OnStandardInvoiceCreated checks for existing CurrentRealizationRunID and returns billing.ValidationError if one exists. OnMutableStandardLinesDeleted requires run not to have Payment or InvoiceUsage already set. |
| `linemapper.go` | populateUsageBasedStandardLineFromRun and helpers: maps RealizationRun to billing.StandardLine fields including MeteredQuantity, Quantity (post-discount), CreditsApplied, DetailedLines, Totals. | Requires run.DetailedLines to be expanded (mo.Some); returns error otherwise. Projected totals must equal run.Totals after rounding — test this invariant when modifying credit application logic. |

## Anti-Patterns

- Calling Adapter methods or state-machine actions directly without going through withLockedCharge — mutating charge state outside the lock+transaction window risks partial writes and lost realizations.
- Adding DB/Ent adapter calls inside the rating or run sub-packages — persistence is exclusively the caller's (this package's) responsibility.
- Constructing StateMachineConfig without calling Validate() first — missing CurrencyCalculator or unexpanded Customer will panic at runtime during rating or credit allocation.
- Firing a trigger without calling AdvanceUntilStateStable afterward in LineEngine hooks — the charge may remain in a transient state when the run result is read by populateUsageBasedStandardLineFromRun.
- Calling getStateMachineConfigForPatch in hot loops — it re-fetches customerOverride and featureMeters from the DB on every call; batch or cache these in callers that process multiple charges.

## Decisions

- **Settlement mode selects the entire state graph at construction time via newStateMachine.** — CreditOnly and CreditThenInvoice have incompatible state graphs; a single unified machine would require complex conditional branches that are harder to test and reason about independently.
- **withLockedCharge always re-fetches the charge with ExpandRealizations inside the advisory lock.** — Realizations are the primary mutable state during advancement; stale data without the lock would produce incorrect credit allocation and duplicate run creation under concurrency.
- **LineEngine is a thin adapter struct over the service rather than a separate top-level type.** — It needs access to rater, runs, adapter, and locker — all owned by service — so embedding via GetLineEngine() avoids duplicating dependency wiring and keeps the LineEngine construction path singular.

## Example: Advancing a usage-based charge through its state machine with locking and settlement-mode dispatch (triggers.go pattern)

```
func (s *service) AdvanceCharge(ctx context.Context, input usagebased.AdvanceChargeInput) (*usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) {
		featureMeter, err := charge.ResolveFeatureMeter(input.FeatureMeters)
		if err != nil { return nil, fmt.Errorf("get feature meter: %w", err) }
		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil { return nil, fmt.Errorf("get currency calculator: %w", err) }
		stateMachine, err := s.newStateMachine(StateMachineConfig{
			Charge: charge, Adapter: s.adapter, Rater: s.rater, Runs: s.runs,
			CustomerOverride: input.CustomerOverride, FeatureMeter: featureMeter,
			CurrencyCalculator: currencyCalculator,
		})
		if err != nil { return nil, fmt.Errorf("new state machine: %w", err) }
// ...
```

<!-- archie:ai-end -->
