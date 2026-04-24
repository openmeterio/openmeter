# service

<!-- archie:ai-start -->

> Implements the usagebased.Service interface: charge creation, state-machine-driven advancement, patch triggering, real-time usage queries, and billing.LineEngine integration. It composes the rating (ClickHouse queries) and run (realization mechanics) sub-packages and routes each charge to the correct settlement-mode state machine (CreditOnly or CreditThenInvoice).

## Patterns

**Config-struct constructor with Validate()** — Every exported constructor (New, newStateMachineBase, NewCreditsOnlyStateMachine, NewCreditThenInvoiceStateMachine) accepts a typed Config/StateMachineConfig and calls config.Validate() before allocating; missing deps produce errors.Join output. (`func New(config Config) (usagebased.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**withLockedCharge wraps every mutating operation** — AdvanceCharge and TriggerPatch always call withLockedCharge, which opens a transaction via transaction.Run, acquires a pg advisory lock via s.locker.LockForTX, then fetches the charge with ExpandRealizations before delegating to the callback. (`return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) { ... })`)
**Settlement-mode dispatch in newStateMachine** — triggers.go newStateMachine switches on charge.Intent.SettlementMode: CreditOnlySettlementMode → CreditsOnlyStateMachine, CreditThenInvoiceSettlementMode → CreditThenInvoiceStateMachine; unsupported modes return GenericNotImplementedError. (`switch config.Charge.Intent.SettlementMode { case productcatalog.CreditOnlySettlementMode: ... case productcatalog.CreditThenInvoiceSettlementMode: ... default: return nil, models.NewGenericNotImplementedError(...) }`)
**FireAndActivate + AdvanceUntilStateStable pattern** — Lifecycle hooks in lineengine.go always call stateMachine.AdvanceUntilStateStable before and after firing invoice-created triggers so all auto-transitions complete before the run result is read. (`if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil { ... } if err := stateMachine.FireAndActivate(ctx, trigger, invoiceCreatedInput{...}); err != nil { ... } if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil { ... }`)
**ensureDetailedLinesLoadedForRating before rating calls** — statemachine.go ensureDetailedLinesLoadedForRating must be called before GetDetailedLinesForUsage when prior runs exist; it lazily fetches all run DetailedLines via Adapter.FetchDetailedLines and mutates s.Charge in place. (`if err := s.ensureDetailedLinesLoadedForRating(ctx); err != nil { return err }`)
**ignoreMinimumCommitment on partial runs** — ignoreMinimumCommitmentForRunType returns true for RealizationRunTypePartialInvoice and false for RealizationRunTypeFinalRealization; pass this flag to CreateRatedRun and GetDetailedLinesForUsage for correctness. (`IgnoreMinimumCommitment: ignoreMinimumCommitmentForRunType(runType)`)
**LineEngine registered via GetLineEngine()** — service.go exposes GetLineEngine() returning &LineEngine{service: s}; callers (billing.Service) register this engine with RegisterLineEngine(billing.LineEngineTypeChargeUsageBased, ...). LineEngine must not be constructed directly. (`func (s *service) GetLineEngine() billing.LineEngine { return &LineEngine{service: s} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Constructor and struct definition. Composes usagebasedrating.Service and usagebasedrun.Service. Exposes GetLineEngine(). | All Config fields are required; missing any causes Validate() to return an error. Do not add business logic here. |
| `triggers.go` | AdvanceCharge and TriggerPatch entry points; withLockedCharge helper; newStateMachine settlement-mode dispatch; getStateMachineConfigForPatch fetches customerOverride + featureMeter. | Every mutating operation must go through withLockedCharge (transaction + advisory lock). getStateMachineConfigForPatch re-fetches customer and feature on every call — avoid calling it in hot loops. |
| `statemachine.go` | Base stateMachine struct embedding chargestatemachine.Machine; shared guards (IsInsideServicePeriod, IsAfterServicePeriod, IsAfterCollectionPeriod); AdvanceAfter setters; ensureDetailedLinesLoadedForRating. | IsAfterCollectionPeriod silently returns false on error (logging only) — don't rely on it for critical decisions. ensureDetailedLinesLoadedForRating mutates s.Charge in place. |
| `creditheninvoice.go` | CreditThenInvoiceStateMachine: full partial+final invoice state graph, StartPartialInvoiceRun/StartFinalInvoiceRun, SnapshotInvoiceUsage, FinalizeInvoiceRun. | SnapshotInvoiceUsage reads CurrentRealizationRunID — fail fast if nil. FinalizeInvoiceRun clears CurrentRealizationRunID and AdvanceAfter after BookAccruedInvoiceUsage. |
| `creditsonly.go` | CreditsOnlyStateMachine: simpler state graph with no invoice steps, final realization only. DeleteCharge corrects credits on CreditRefundPolicyCorrect. | CreditAllocationExact (not CreditAllocationAvailable) is used for credits-only charges. storedAtOffset is clock.Now() minus InternalCollectionPeriod. |
| `lineengine.go` | billing.LineEngine implementation: SplitGatheringLine, BuildStandardInvoiceLines, OnStandardInvoiceCreated, OnCollectionCompleted, OnInvoiceIssued, OnPaymentAuthorized, OnPaymentSettled, CalculateLines. | SplitGatheringLine clears ChildUniqueReferenceID on both halves. OnStandardInvoiceCreated checks for existing CurrentRealizationRunID and returns billing.ValidationError if one exists. |
| `payments.go` | recordRunPayments, recordPaymentAuthorized, recordPaymentSettled helpers. Fires TriggerAllPaymentsSettled when areAllInvoicedRunsSettled returns true. | areAllInvoicedRunsSettled requires at least one FinalRealization run with InvoiceUsage; returns false if any invoiced run has nil Payment or non-Settled status. |

## Anti-Patterns

- Calling Adapter methods directly from a state machine action without going through withLockedCharge — mutating state outside the lock+transaction window risks partial writes.
- Adding DB or Ent adapter calls inside the rating or run sub-packages — persistence is the caller's responsibility.
- Constructing StateMachineConfig without calling Validate() first — missing CurrencyCalculator or expanded Customer will panic at runtime.
- Firing a trigger without calling AdvanceUntilStateStable afterward in LineEngine hooks — the charge may remain in a transient state when the run result is read.
- Reading currentRun.MeterValue or currentRun.CreditsAllocated before SnapshotInvoiceUsage completes — data will be zero/empty.

## Decisions

- **Settlement mode selects the entire state graph at construction time via newStateMachine.** — CreditOnly and CreditThenInvoice have incompatible state graphs; a single unified machine would require complex conditional branches that are harder to test and reason about.
- **withLockedCharge always re-fetches the charge with ExpandRealizations inside the lock.** — Realizations are the primary mutable state during advancement; stale data without the lock would produce incorrect credit allocation and run creation.
- **LineEngine is a thin adapter over the service rather than a separate top-level type.** — It needs access to rater, runs, adapter, and locker — all owned by service — so embedding avoids duplicating dependency wiring.

## Example: Advancing a usage-based charge through its state machine with proper locking and settlement-mode dispatch

```
// triggers.go
func (s *service) AdvanceCharge(ctx context.Context, input usagebased.AdvanceChargeInput) (*usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) {
		featureMeter, err := charge.ResolveFeatureMeter(input.FeatureMeters)
		// ...
		stateMachine, err := s.newStateMachine(StateMachineConfig{
			Charge: charge, Adapter: s.adapter, Rater: s.rater, Runs: s.runs,
			CustomerOverride: input.CustomerOverride, FeatureMeter: featureMeter,
			CurrencyCalculator: currencyCalculator,
		})
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
// ...
```

<!-- archie:ai-end -->
