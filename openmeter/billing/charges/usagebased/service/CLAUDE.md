# service

<!-- archie:ai-start -->

> The orchestration layer for usage-based billing charges: it owns the per-settlement-mode state machines (credit_only, credit_then_invoice), the billing.LineEngine implementation, and the charges.Service entrypoints (Create, AdvanceCharge, TriggerPatch, GetByID, GetCurrentTotals). It decides WHICH triggers fire and WHICH status to enter, delegating rating to the rating/ subpackage and run mechanics (persistence, credits, payments) to the run/ subpackage.

## Patterns

**Config-validated New with composed sub-services** — New(Config) validates every dependency is non-nil, then constructs the rating Service (usagebasedrating.New) and run Service (usagebasedrun.New) internally and stores them as s.rater / s.runs. Public methods are on *service. (`func New(config Config) (usagebased.Service, error) { if err := config.Validate(); err != nil { return nil, err }; rater, err := usagebasedrating.New(...); runs, err := usagebasedrun.New(...); return &service{adapter: config.Adapter, rater: rater, runs: runs, ...} }`)
**Settlement-mode dispatch to a state machine** — newStateMachine switches on config.Charge.Intent.SettlementMode: CreditOnlySettlementMode -> NewCreditsOnlyStateMachine, CreditThenInvoiceSettlementMode -> NewCreditThenInvoiceStateMachine, default -> NewGenericNotImplementedError. Each state-machine constructor re-asserts the settlement mode and calls configureStates(). (`switch config.Charge.Intent.SettlementMode { case productcatalog.CreditOnlySettlementMode: return NewCreditsOnlyStateMachine(config) ... default: return nil, models.NewGenericNotImplementedError(...) }`)
**Declarative state config via stateless library + meta.Trigger* constants** — configureStates() builds the machine with s.Configure(Status).Permit(meta.Trigger, NextStatus, guard).OnActive/OnEntry(...). Guards use statelessx.BoolFn / WithParameters / AllOf. Status and Trigger values live in usagebased and meta packages — never invent local statuses. (`s.Configure(usagebased.StatusActive).Permit(meta.TriggerFinalInvoiceCreated, usagebased.StatusActiveFinalRealizationStarted, statelessx.BoolFn(s.IsAfterServicePeriod)).OnActive(statelessx.AllOf(s.SyncFeatureIDFromFeatureMeter, s.AdvanceAfterServicePeriodTo))`)
**Input.Validate() then transaction.Run(s.adapter, ...)** — Every public service method validates its Input first, then wraps the body in transaction.Run(ctx, s.adapter, fn). Mutating-by-id flows additionally go through withLockedCharge which acquires charges.NewLockKeyForCharge + locker.LockForTX before refetching the charge. (`return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) { ... return stateMachine.AdvanceUntilStateStable(ctx) })`)
**LineEngine bridges billing invoice lifecycle into state-machine triggers** — LineEngine implements billing.LineEngine (var _ billing.LineEngine = (*LineEngine)(nil)) returning LineEngineTypeChargeUsageBased. Its callbacks (OnStandardInvoiceCreated, OnCollectionCompleted, OnMutableStandardLinesDeleted) build a state machine per line via newStateMachineForStandardLine, fire the resolved trigger, AdvanceUntilStateStable, then populateUsageBasedStandardLineFromRun + stdLine.Validate(). (`trigger := resolveInvoiceCreatedTrigger(stateMachine.GetCharge(), stdLine.Period); stateMachine.FireAndActivate(ctx, trigger, invoiceCreatedInput{LineID: stdLine.ID, InvoiceID: input.Invoice.ID, ServicePeriodTo: stdLine.Period.To})`)
**Invoice mutations emitted as deferred invoiceupdater.Patch, not direct writes** — State-machine actions (DeleteCharge, ExtendCharge/ShrinkCharge handlers) never mutate invoices directly; they call s.AddInvoicePatch(invoiceupdater.New*Patch(...)). TriggerPatch drains them via stateMachine.DrainInvoicePatches() into result.InvoicePatches for the caller to apply. Voided/already-deleted runs are skipped (run.IsVoidedBillingHistory()). (`s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(billing.LineID{Namespace: s.Charge.Namespace, ID: *run.LineID}, *run.InvoiceID))`)
**Realization-run lifecycle delegated to s.runs / s.rater** — Status actions never call ent/ledger/credit/streaming directly. StartFinalRealizationRun -> s.Runs.CreateRatedRun; FinalizeRealizationRun -> s.Rater.GetDetailedRatingForUsage + s.Runs.ReconcileCredits + s.Adapter.UpsertRunDetailedLines/UpdateRealizationRun; credit correction on delete -> s.Runs.CorrectAllCredits. Totals are RoundToPrecision(CurrencyCalculator) before compare. (`result, err := s.Runs.CreateRatedRun(ctx, usagebasedrun.CreateRatedRunInput{Charge: s.Charge, Type: usagebased.RealizationRunTypeFinalRealization, CreditAllocation: usagebasedrun.CreditAllocationExact, NoFiatTransactionRequired: true, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config/New/service struct; composes rater+runs sub-services; GetLineEngine() returns &LineEngine{service: s}. | All deps are required (Validate rejects nil). Do not construct service{} directly; rater and runs must be built through New so their own Config validation runs. |
| `triggers.go` | AdvanceCharge, TriggerPatch, newStateMachine dispatch, getStateMachineConfigForPatch, withLockedCharge. | newStateMachine is the ONLY place settlement-mode->machine mapping belongs. withLockedCharge always re-fetches the charge with meta.ExpandRealizations after locking — don't trust a charge passed in from outside the lock. |
| `creditheninvoice.go` | CreditThenInvoiceStateMachine: full partial-invoice + final-realization + payment-settlement status graph; Extend/Shrink/Delete patch handlers. | Immutable boundaries (StatusActive*Issuing/Completed) MUST use UnsupportedExtendOperation/UnsupportedShrinkOperation (NewGenericPreConditionFailedError), not the real handlers. Shrinking past an invoice-backed run that is not line/invoice-backed is a precondition failure. |
| `creditsonly.go` | CreditsOnlyStateMachine: simpler credit-only graph; StartFinalRealizationRun / FinalizeRealizationRun; DeleteCharge honours CreditRefundPolicyCorrect. | Credit-only charges set NoFiatTransactionRequired: true and never touch the invoicing stack (Create short-circuits before gatheringLineFromUsageBasedCharge for CreditOnlySettlementMode). |
| `statemachine.go` | Shared stateMachine base (embeds chargestatemachine.Machine), StateMachineConfig+Validate, newStateMachineBase, guard/advance helpers (IsAfterServicePeriod, AdvanceAfterCollectionPeriodEnd, getFinalRunStoredAtLT). | StateMachineConfig.Validate requires expanded CustomerOverride.Customer, a valid MergedProfile, FeatureMeter.Meter, and CurrencyCalculator. storedAtLT for final runs is derived from MergedProfile.WorkflowConfig.Collection.Interval. |
| `lineengine.go` | LineEngine: billing.LineEngine impl — Split/Build/OnStandardInvoiceCreated/OnCollectionCompleted/OnMutableStandardLinesDeleted/OnUnsupportedCreditNote + gathering preview. | Only CreditThenInvoiceSettlementMode is supported on the invoicing path; other modes error. Guards before mutation: CurrentRealizationRunID must be nil (ErrActiveRealizationRunAlreadyExists), run must belong to input.Invoice.ID, and run.DeletedAt/Payment/InvoiceUsage gate deletion. |
| `linemapper.go` | populateUsageBasedStandardLineFromRun + mapUsageBasedDetailedLines: projects a RealizationRun onto a billing.StandardLine (quantities, credits, detailed lines, totals). | Run detailed lines must be expanded (mo.Some) or it errors. Mapped line totals are asserted equal to run.Totals.RoundToPrecision — a mismatch is a hard error. Uses mutator.ApplyUsageDiscount for net vs raw metered quantity. |
| `create.go` | service.Create: bulk-creates charges (s.adapter.CreateCharges), registers them with metaAdapter.RegisterCharges (ChargeTypeUsageBased), builds gathering lines per charge. | CreditOnlySettlementMode charges return early with no GatheringLineToCreate. RatingEngine is chosen at create time via s.rater.GetPreferredRatingEngineFor(intent) and frozen into the charge intent. |

## Anti-Patterns

- Putting settlement-mode->state-machine selection anywhere other than newStateMachine in triggers.go, or hardcoding a machine type in a handler.
- Calling ent/ledger/credit/streaming/rating directly from a status action instead of going through s.runs, s.rater, or s.adapter (rating/run decisions live in the subpackages).
- Mutating invoices directly from the state machine instead of emitting invoiceupdater.Patch via AddInvoicePatch / DrainInvoicePatches.
- Letting Extend/Shrink run on immutable statuses (StatusActive*Issuing/Completed) — those must map to UnsupportedExtend/ShrinkOperation, otherwise invoice-issued callbacks lose ownership of the state.
- Comparing run/line totals or credit amounts without RoundToPrecision(CurrencyCalculator), or projecting a run to a standard line without re-validating that line totals equal run totals.

## Decisions

- **Split the orchestration service (this folder) from rating/ and run/ subpackages so this layer only makes state-machine decisions while the subpackages own rating and run mechanics.** — Keeps the trigger/status logic auditable in one place and lets credits, payments, and rating be tested and reasoned about independently of the lifecycle graph.
- **Each settlement mode is its own state machine (CreditsOnly vs CreditThenInvoice) sharing a common stateMachine base rather than one machine with mode-conditional branches.** — Credit-only charges never invoice, while credit-then-invoice has partial-invoice, final-realization, and payment-settlement phases; separate configureStates() graphs keep each lifecycle explicit and prevent illegal transitions.
- **Invoice side effects are emitted as deferred invoiceupdater.Patch objects drained by the caller, not applied inline.** — The charge transaction and the billing invoice mutation are owned by different layers; patches let billing apply line create/delete/update against mutable vs immutable invoices appropriately (e.g. credit-note vs hard delete).

## Example: Bridging a billing invoice-created event into a charge state-machine trigger and projecting the resulting run back onto the line

```
func (e *LineEngine) OnStandardInvoiceCreated(ctx context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil { return nil, fmt.Errorf("validating input: %w", err) }
	return slicesx.MapWithErr(input.Lines, func(stdLine *billing.StandardLine) (*billing.StandardLine, error) {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil { return nil, err }
		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil { return nil, err }
		trigger := resolveInvoiceCreatedTrigger(stateMachine.GetCharge(), stdLine.Period)
		if stateMachine.GetCharge().State.CurrentRealizationRunID != nil {
			return nil, billing.ValidationError{Err: fmt.Errorf("line[%s]: %w", stdLine.ID, usagebased.ErrActiveRealizationRunAlreadyExists)}
		}
		if err := stateMachine.FireAndActivate(ctx, trigger, invoiceCreatedInput{LineID: stdLine.ID, InvoiceID: input.Invoice.ID, ServicePeriodTo: stdLine.Period.To}); err != nil { return nil, err }
		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil { return nil, err }
		charge := stateMachine.GetCharge()
		currentRun, err := charge.GetCurrentRealizationRun()
		if err != nil { return nil, err }
// ...
```

<!-- archie:ai-end -->
