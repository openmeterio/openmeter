# service

<!-- archie:ai-start -->

> Implements the usagebased.Service interface — charge creation, state-machine-driven advancement, patch triggering, real-time usage queries, and billing.LineEngine integration. It composes two sub-packages (rating/ for ClickHouse quantity queries, run/ for realization mechanics) and dispatches each charge to the correct settlement-mode state machine (CreditsOnly or CreditThenInvoice).

## Patterns

**Config-struct constructor with Validate()** — Every exported constructor (New, newStateMachineBase, NewCreditsOnlyStateMachine, NewCreditThenInvoiceStateMachine) accepts a typed Config/StateMachineConfig and calls config.Validate() via errors.Join before allocating; missing deps produce a joined error list. (`func New(config Config) (usagebased.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**withLockedCharge wraps every mutating operation** — AdvanceCharge and TriggerPatch call withLockedCharge, which opens transaction.Run, acquires a pg advisory lock via s.locker.LockForTX using charges.NewLockKeyForCharge, then re-fetches the charge with ExpandRealizations before the callback. Never mutate charge state outside this wrapper. (`return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) { ... })`)
**Settlement-mode dispatch in newStateMachine** — triggers.go switches on charge.Intent.SettlementMode: CreditOnlySettlementMode -> CreditsOnlyStateMachine, CreditThenInvoiceSettlementMode -> CreditThenInvoiceStateMachine; unsupported modes return models.NewGenericNotImplementedError. (`switch config.Charge.Intent.SettlementMode { case productcatalog.CreditOnlySettlementMode: ... case productcatalog.CreditThenInvoiceSettlementMode: ... default: return nil, models.NewGenericNotImplementedError(...) }`)
**FireAndActivate + AdvanceUntilStateStable bookend in LineEngine hooks** — LineEngine hooks (OnStandardInvoiceCreated, OnCollectionCompleted, OnInvoiceIssued) call AdvanceUntilStateStable before firing the invoice trigger and again after, so all auto-transitions complete before the run result is read. (`stateMachine.AdvanceUntilStateStable(ctx); stateMachine.FireAndActivate(ctx, trigger, input); stateMachine.AdvanceUntilStateStable(ctx)`)
**GetLineEngine() is the sole LineEngine construction path** — service.go exposes GetLineEngine() returning &LineEngine{service: s}; billing.Service registers it via RegisterLineEngine(billing.LineEngineTypeChargeUsageBased, ...). LineEngine must not be constructed directly — it needs rater, runs, adapter, and locker owned by service. (`func (s *service) GetLineEngine() billing.LineEngine { return &LineEngine{service: s} }`)
**Parallel rating with semaphore in expandChargesUsage** — get.go uses semaphore.NewWeighted(defaultMaxParallelRatingsPerRequest=5) + sync.WaitGroup for concurrent ClickHouse rating, with per-goroutine panic recovery and errors.Join on all channel errors. (`sem := semaphore.NewWeighted(int64(defaultMaxParallelRatingsPerRequest)); sem.Acquire(ctx,1); defer sem.Release(1)`)
**Projected line totals must equal run totals** — linemapper.go populateUsageBasedStandardLineFromRun projects run DetailedLines onto stdLine and verifies stdLine.Totals equals run.Totals after RoundToPrecision; a mismatch is a fatal error. (`if !stdLine.Totals.Equal(expectedTotals) { return fmt.Errorf("projected line totals do not match run totals ...") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Constructor and struct definition; composes rating.Service and run.Service internally and exposes GetLineEngine(). All Config fields are required. | No business logic here; every Config field must be non-nil or Validate() returns a joined error — add new fields with a matching nil check. |
| `triggers.go` | AdvanceCharge and TriggerPatch entry points; withLockedCharge helper (transaction + advisory lock + re-fetch); newStateMachine settlement-mode dispatch; getStateMachineConfigForPatch. | Every mutating path must go through withLockedCharge. getStateMachineConfigForPatch re-fetches customerOverride + featureMeter on every call — avoid in hot loops. |
| `statemachine.go` | Base stateMachine struct embedding chargestatemachine.Machine; shared guards (IsInsideServicePeriod, IsAfterServicePeriod, IsAfterCollectionPeriod), AdvanceAfter setters, ensureDetailedLinesLoadedForRating. | IsAfterCollectionPeriod silently returns false on error (logs only). ensureDetailedLinesLoadedForRating mutates s.Charge in place. |
| `creditheninvoice.go` | CreditThenInvoiceStateMachine: full partial+final invoice state graph, StartPartialInvoiceRun, StartFinalInvoiceRun, SnapshotInvoiceUsage, FinalizeInvoiceRun, Extend/Shrink/Delete handlers. | SnapshotInvoiceUsage reads CurrentRealizationRunID — fail fast if nil. Extend/Shrink in issuing/completed states return GenericPreConditionFailedError. |
| `creditsonly.go` | CreditsOnlyStateMachine: simpler graph with no invoice steps, final realization only; DeleteCharge corrects credits on CreditRefundPolicyCorrect; uses CreditAllocationExact. | storedAtLT is clock.Now() minus InternalCollectionPeriod. Uses CreditAllocationExact (not CreditAllocationAvailable). |
| `lineengine.go` | billing.LineEngine impl: SplitGatheringLine, BuildStandardInvoiceLines, OnStandardInvoiceCreated, OnCollectionCompleted, OnInvoiceIssued, OnPaymentAuthorized, OnPaymentSettled, OnMutableStandardLinesDeleted. | SplitGatheringLine clears ChildUniqueReferenceID on both halves. OnStandardInvoiceCreated returns billing.ValidationError if a CurrentRealizationRunID already exists. OnMutableStandardLinesDeleted requires no Payment/InvoiceUsage on the run. |
| `linemapper.go` | populateUsageBasedStandardLineFromRun: maps RealizationRun -> billing.StandardLine (MeteredQuantity, post-discount Quantity, CreditsApplied, DetailedLines, Totals). | Requires run.DetailedLines expanded (mo.Some); projected totals must equal run.Totals after rounding — test this invariant when changing credit application. |
| `create.go` | Create: bulk-creates charges via adapter.CreateCharges, registers them in meta, and builds gathering lines (gatheringLineFromUsageBasedCharge) for non credit-only settlement. | CreditOnlySettlementMode charges return early with no gathering line; always Clone() Annotations/Metadata before placing into the gathering line. |

## Anti-Patterns

- Calling Adapter methods or state-machine actions directly without withLockedCharge — mutating charge state outside the lock+transaction risks partial writes and lost realizations.
- Adding DB/Ent adapter calls inside the rating/ or run/ sub-packages — persistence is exclusively this package's (the caller's) responsibility.
- Constructing StateMachineConfig without Validate() first — missing CurrencyCalculator or unexpanded Customer panics during rating or credit allocation.
- Firing a trigger in a LineEngine hook without AdvanceUntilStateStable afterward — the charge may stay in a transient state when populateUsageBasedStandardLineFromRun reads the run result.
- Calling getStateMachineConfigForPatch in hot loops — it re-fetches customerOverride and featureMeters per call; batch or cache in callers processing many charges.

## Decisions

- **Settlement mode selects the entire state graph at construction time via newStateMachine.** — CreditOnly and CreditThenInvoice have incompatible state graphs; a single unified machine would need complex conditional branches that are harder to test and reason about independently.
- **withLockedCharge always re-fetches the charge with ExpandRealizations inside the advisory lock.** — Realizations are the primary mutable state during advancement; stale data without the lock would produce incorrect credit allocation and duplicate run creation under concurrency.
- **LineEngine is a thin adapter struct over the service rather than a separate top-level type.** — It needs rater, runs, adapter, and locker — all owned by service — so embedding via GetLineEngine() avoids duplicating dependency wiring and keeps construction singular.

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
