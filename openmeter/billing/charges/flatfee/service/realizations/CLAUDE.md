# realizations

<!-- archie:ai-start -->

> Owns flat-fee realization mechanics: credit allocation, credit correction, and detailed-line/run-totals persistence for the credit_then_invoice, credits-only, and invoice-accrued lifecycle modes. Strictly a mechanics layer — it must NOT make state-machine decisions (those live in the flatfee state machine that calls it).

## Patterns

**Input struct + Validate() per operation** — Every public Service method takes one named Input struct exposing a Validate() error that aggregates field errors via errors.Join and returns models.NewNillableGenericValidationError. The method calls in.Validate() first and returns the zero Result on failure. (`func (i ReconcileCreditRealizationsInput) Validate() error { var errs []error; if err := i.Charge.Validate(); err != nil { errs = append(errs, fmt.Errorf("charge: %w", err)) }; ... return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Named Result structs** — Methods return a dedicated Result struct (e.g. ReconcileCreditRealizationsResult, AccrueInvoiceUsageResult) carrying the persisted run, delta, and creditrealization.Realizations — never bare tuples of domain values. (`type StartCreditThenInvoiceRunResult struct { Run flatfee.RealizationRun }`)
**Wrap persistence in transaction.Run** — Any method that performs multiple adapter writes (create run, create credit allocations, upsert detailed lines, update run totals) wraps the body in transaction.Run(ctx, s.adapter, func(ctx)...). AllocateCreditsOnly, StartCreditThenInvoiceRun, ReconcileStandardLineToIntent, and AccrueInvoiceUsage all do this; pure-compute BuildCreditThenInvoiceGatheringPreviewRun does not (no ctx, no writes). (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (AccrueInvoiceUsageResult, error) { ... })`)
**Round through CurrencyCalculator before compare/persist** — Amounts are passed through in.CurrencyCalculator.RoundToPrecision before delta computation, equality checks, and persistence. Deltas drive a delta.IsPositive/IsNegative/IsZero switch (positive=allocate, negative=correct lineage, zero=no-op). (`in.TargetAmount = in.CurrencyCalculator.RoundToPrecision(in.TargetAmount); delta := in.CurrencyCalculator.RoundToPrecision(in.TargetAmount.Sub(currentAmount))`)
**Delegate credit math to the flatfee Handler, persist via Adapter** — Credit allocation/correction decisions go through s.handler.OnAllocateCredits / OnCorrectCreditAllocations; persistence (CreateCurrentRun, CreateCreditAllocations, UpsertDetailedLines, UpdateRealizationRun, CreateInvoicedUsage) goes through s.adapter. The createCreditAllocations helper always pairs adapter.CreateCreditAllocations with lineage.CreateInitialLineages + PersistCorrectionLineageSegments. (`realizations, err := s.adapter.CreateCreditAllocations(ctx, runID, creditAllocations); ... s.lineage.CreateInitialLineages(ctx, ...)`)
**Rate line, then apply credits to derive detailed lines/totals** — rateFlatFeeLine clones the line, clears CreditsApplied and split metadata, calls ratingService.GenerateDetailedLines(WithCreditsMutatorDisabled), merges via invoicecalc.MergeGeneratedDetailedLines. applyCreditsToFlatFeeLine then re-applies CreditsApplied and recomputes Totals. Both validate before returning. (`line, err := rateFlatFeeLine(in.Line, s.ratingService); mappedLine, err := applyCreditsToFlatFeeLine(*line, creditsApplied, currencyCalculator)`)
**Tag credit allocations with LineID** — Allocation inputs from the handler are mapped to stamp allocation.LineID = run/line ID before persistence so credit realizations are linked to the originating invoice line. (`creditAllocationsWithLineID := creditrealization.CreateAllocationInputs(lo.Map(creditAllocations, func(a creditrealization.CreateAllocationInput, _ int) creditrealization.CreateAllocationInput { a.LineID = lo.ToPtr(in.Line.ID); return a }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct (adapter flatfee.Adapter, handler flatfee.Handler, lineage lineage.Service, ratingService rating.Service), Config+Validate, New constructor, and the shared createCreditAllocations helper. | Service must NOT make state-machine decisions (documented contract). createCreditAllocations always runs CreateInitialLineages AND PersistCorrectionLineageSegments — never skip the lineage calls when adding a new credit path. |
| `correct.go` | ReconcileCredits (delta-driven allocate/correct/no-op) and CorrectAllCredits — adjust a run's credit realizations to a target or fully reverse them, loading active lineage segments via lineage.LoadActiveSegmentsByRealizationID. | Negative delta must correct existing lineage (Run.CreditRealizations.Correct), not create unrelated negative rows. TargetAmount is rounded before Validate. |
| `creditsonly.go` | AllocateCreditsOnly — credit allocation with no invoice line; updates the current run totals (CreditsTotal=allocated, Total=Zero) inside a transaction. | Asserts allocated.Equal(in.Amount) and returns NewGenericValidationError on mismatch. Requires Charge.Realizations.CurrentRun != nil. |
| `credittheninvoice.go` | StartCreditThenInvoiceRun (create mutable CTI run + allocate credits) and ReconcileStandardLineToIntent (re-sync a mutable line+run after intent change). Hosts the shared rateFlatFeeLine/applyCreditsToFlatFeeLine helpers. | Reconcile uses the rebuilt LINE period for both credit allocation and the run update (run.ServicePeriod = in.Line.Period) so ledger and invoice describe the same window. rateFlatFeeLine clears SplitLineGroupID/SplitLineHierarchy so the flat pricer doesn't skip a billable run. |
| `invoiceaccrued.go` | AccrueInvoiceUsage — books invoice-accrued usage via handler.OnInvoiceUsageAccrued, persists invoicedusage.AccruedUsage, then marks the run Immutable. | Validate rejects when CurrentRun.AccruedUsage already set (no double-accrual) and enforces run/line/invoice ID matches. Skips ledger work when line Totals.Total IsZero but still sets the run Immutable. |
| `preview.go` | BuildCreditThenInvoiceGatheringPreviewRun — pure-compute preview run shape for gathering-invoice expansion; no ctx, no persistence, no credit allocation. | Must stay side-effect-free (get/list expansion). Uses a synthetic run ID 'preview-<lineID>' and only supports CreditThenInvoiceSettlementMode. |

## Anti-Patterns

- Making state-machine / lifecycle decisions here — this Service is mechanics only; decisions belong in the flatfee state machine that calls it.
- Performing multiple adapter writes without wrapping them in transaction.Run(ctx, s.adapter, ...).
- Creating raw negative credit rows for a shrinking amount instead of correcting existing lineage via CreditRealizations.Correct.
- Adding side effects (credit allocation, persistence, ctx) to preview.go — preview must remain pure so get/list expansion stays read-only.
- Skipping CurrencyCalculator.RoundToPrecision before comparing/persisting amounts, or skipping in.Validate() at method entry.

## Decisions

- **Mechanics-only Service with no state-machine logic.** — Keeps credit allocation/correction and lineage persistence reusable across credits-only, credit_then_invoice, and invoice-accrued modes without coupling to lifecycle transitions.
- **Reconciliation drives credits off a delta vs. target amount.** — When a mutable standard line changes amount, a single delta switch (allocate on growth, correct on shrink, no-op on equal) avoids re-deriving the whole credit ledger and preserves lineage.
- **Re-rate line with split metadata cleared and credits mutator disabled, then re-apply credits.** — Flat-fee charges materialize their own billable periods, so subscription split-line metadata must not make the flat pricer skip a billable run; credits are applied as a separate, explicit step.

## Example: Delta-driven credit reconciliation toward a target amount

```
func (s *Service) ReconcileCredits(ctx context.Context, in ReconcileCreditRealizationsInput) (ReconcileCreditRealizationsResult, error) {
	in.TargetAmount = in.CurrencyCalculator.RoundToPrecision(in.TargetAmount)
	if err := in.Validate(); err != nil {
		return ReconcileCreditRealizationsResult{}, err
	}
	currentAmount := in.CurrencyCalculator.RoundToPrecision(in.Run.CreditRealizations.Sum())
	delta := in.CurrencyCalculator.RoundToPrecision(in.TargetAmount.Sub(currentAmount))
	result := ReconcileCreditRealizationsResult{Delta: delta}
	switch {
	case delta.IsPositive():
		creditAllocations, err := s.handler.OnAllocateCredits(ctx, flatfee.OnAllocateCreditsInput{Charge: in.Charge, ServicePeriod: in.Run.ServicePeriod, BookedAt: in.AllocateAt, PreTaxAmountToAllocate: delta})
		if err != nil {
			return ReconcileCreditRealizationsResult{}, fmt.Errorf("allocate credits for flat fee: %w", err)
		}
		// ... stamp LineID, s.createCreditAllocations(...)
// ...
```

<!-- archie:ai-end -->
