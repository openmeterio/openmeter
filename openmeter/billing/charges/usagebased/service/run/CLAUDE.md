# run

<!-- archie:ai-start -->

> Owns the mechanics of usage-based realization runs: rating snapshots, run persistence, credit allocation/correction, credit-realization lineage, invoice-usage booking, and payment authorize/settle. It deliberately does NOT make state-machine decisions (which triggers to fire, which status to enter) — those belong to the caller in usagebased/service.

## Patterns

**Service struct with validated Config + New** — Service holds exactly four collaborators (adapter usagebased.Adapter, rater usagebasedrating.Service, handler usagebased.Handler, lineage lineage.Service). Config.Validate() returns errors.Join over nil-checks; New(config) validates before constructing. Never add unvalidated deps. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &Service{adapter: config.Adapter, ...}, nil }`)
**Input struct + Validate() per public method** — Every public method takes a single typed Input struct with a Validate() method called first thing in the method body. Inputs embed Charge/Run and call their .Validate(). Result is returned as a typed *Result struct, never bare values. (`func (s *Service) CreateRatedRun(ctx, in CreateRatedRunInput) (CreateRatedRunResult, error) { if err := in.Validate(); err != nil { return CreateRatedRunResult{}, err }; ... }`)
**Decisions delegated to handler, persistence to adapter** — Side-effecting credit/ledger decisions go through s.handler (OnCreditsOnlyUsageAccrued, OnInvoiceUsageAccrued, OnPaymentAuthorized/Settled, OnCreditsOnlyUsageAccruedCorrection); DB writes go through s.adapter (CreateRealizationRun, UpdateRealizationRun, CreateRunCreditRealization, CreateRunInvoicedUsage, CreateRunPayment, UpsertRunDetailedLines). Run does not call ent or ledger directly. (`creditAllocations, err := s.handler.OnCreditsOnlyUsageAccrued(ctx, usagebased.CreditsOnlyUsageAccruedInput{...})`)
**CurrencyCalculator.RoundToPrecision before decimal compares** — All target/current/delta amounts are rounded via in.CurrencyCalculator.RoundToPrecision(...) before IsZero/IsPositive/IsNegative/Equal branching. Skipping rounding causes spurious non-zero deltas. (`delta := in.CurrencyCalculator.RoundToPrecision(in.TargetAmount.Sub(currentAmount)); switch { case delta.IsPositive(): ... }`)
**Credit reconciliation as signed-delta switch** — ReconcileCredits computes delta = target - current and switches: positive -> allocate(), negative -> CreditsAllocated.Correct(...) producing corrections then createRunCreditRealizations, zero -> no-op. Corrections always preload lineage via lineage.LoadActiveSegmentsByRealizationID. (`case delta.IsNegative(): corrections, err := in.Run.CreditsAllocated.Correct(delta, calc, func(req) {...s.handler.OnCreditsOnlyUsageAccruedCorrection...})`)
**NoFiatTransactionRequired short-circuits fiat paths** — When run.NoFiatTransactionRequired (credits_only or zero total) the payment authorize/settle paths return early without ledger calls, and invoice-usage booking creates AccruedUsage with no LedgerTransaction. Validate() enforces the zero-total <-> NoFiatTransactionRequired invariant. (`if in.Run.NoFiatTransactionRequired { return BookInvoicedPaymentAuthorizedResult{Run: in.Run}, nil }`)
**Lineage created/persisted alongside every credit realization** — createRunCreditRealizations always follows adapter.CreateRunCreditRealization with lineage.CreateInitialLineages and lineage.PersistCorrectionLineageSegments. New credit-writing paths must funnel through this helper, not call the adapter directly. (`realizations, _ := s.adapter.CreateRunCreditRealization(ctx, runID, allocs); s.lineage.CreateInitialLineages(ctx, ...); s.lineage.PersistCorrectionLineageSegments(ctx, ...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service/Config/New plus CreditAllocationMode enum (None/Exact/Available). Docstring spells out the no-state-machine constraint. | Do not add state-machine logic here; keep the four-collaborator shape and Config.Validate nil-checks in sync. |
| `create.go` | CreateRatedRun: rates usage via rater, builds the realization run, optionally allocates credits, recomputes totals/NoFiatTransactionRequired, upserts detailed lines. | Guards against a pre-existing CurrentRealizationRunID in both Input.Validate and createNewRealizationRun; ServicePeriodTo must be within charge.Intent.ServicePeriod (after From, not after To). Negative total -> ErrChargeTotalIsNegative. |
| `credits.go` | allocate() (handler.OnCreditsOnlyUsageAccrued + lineage), createRunCreditRealizations() helper, featuresForLineage(). | Allocated must not exceed AmountToAllocate (ErrCreditAllocationsDoNotMatchTotal); Exact mode requires exact equality. Always round AmountToAllocate first; zero amount short-circuits. |
| `correct.go` | ReconcileCredits (signed-delta allocate/correct) and CorrectAllCredits (reverse all realizations), both preloading lineage segments. | TargetAmount must be zero-or-positive after rounding; corrections drive handler.OnCreditsOnlyUsageAccruedCorrection per CorrectionRequest, not direct ledger writes. |
| `invoice.go` | BookAccruedInvoiceUsage: links a run to a billing.StandardLine and records invoicedusage.AccruedUsage (with ledger txn unless NoFiatTransactionRequired). | Validate enforces run.LineID matches line.ID, run has no existing InvoiceUsage, and the NoFiatTransactionRequired<->zero-total invariant. Non-fiat path requires a non-empty TransactionGroupID. |
| `payment.go` | BookInvoicedPaymentAuthorized and SettleInvoicedPayment: ledger handler calls + adapter Create/UpdateRunPayment, using payment.Status transitions Authorized->Settled. | Settle requires an existing Authorized payment with matching LineID unless NoFiatTransactionRequired. Uses clock.Now() for eventAt — freeze clock in tests. |
| `preview.go` | BuildCreditThenInvoiceGatheringPreviewRun: side-effect-free run shape for gathering-invoice previews; rates usage but persists nothing and allocates no credits. | Only supports CreditThenInvoiceSettlementMode; uses NewNillableGenericValidationError collecting all errs. Synthesizes a preview-<lineID> run ID; do not call adapters here. |
| `payment_test.go` | Validate-focused unit tests with newBookPaymentAuthorizedInput/newSettlePaymentInput/newUsageBasedCharge/newUsageBasedRun fixtures. | Hand-assembled usagebased.Charge/RealizationRun fixtures — keep field names aligned with the domain structs when they change. |

## Anti-Patterns

- Making state-machine decisions here (firing triggers, choosing Status, advancing invoice lifecycle) — that belongs to the caller in usagebased/service.
- Calling ent/ledger/credit directly instead of going through s.adapter / s.handler / s.lineage.
- Comparing alpacadecimal amounts (IsZero/Equal/IsPositive) without first RoundToPrecision via the CurrencyCalculator.
- Writing credit realizations without the matching lineage.CreateInitialLineages + PersistCorrectionLineageSegments calls.
- Creating a second realization run while charge.State.CurrentRealizationRunID is already set.

## Decisions

- **Run service is split from the state-machine service and restricted to run mechanics.** — Keeps rating/persistence/credit/lineage concerns reusable and testable independent of which lifecycle trigger fires; the docstring on Service codifies the boundary.
- **Credit reconciliation is modeled as a signed delta with allocate vs correct branches rather than recompute-and-replace.** — Allows incremental allocation on positive deltas and ledger-correct reversal on negative deltas while preserving lineage continuity.
- **NoFiatTransactionRequired is threaded through invoice/payment paths as an early-return guard with a Validate-enforced zero-total invariant.** — credits_only and zero-total runs must never touch the fiat ledger, and the invariant prevents booking a fiat payment for a zero line or skipping one for a non-zero line.

## Example: Reconcile a run's credits toward a target amount (positive=allocate, negative=correct).

```
func (s *Service) ReconcileCredits(ctx context.Context, in ReconcileCreditRealizationsInput) (ReconcileCreditRealizationsResult, error) {
	in.TargetAmount = in.CurrencyCalculator.RoundToPrecision(in.TargetAmount)
	if err := in.Validate(); err != nil { return ReconcileCreditRealizationsResult{}, err }
	current := in.CurrencyCalculator.RoundToPrecision(in.Run.CreditsAllocated.Sum())
	delta := in.CurrencyCalculator.RoundToPrecision(in.TargetAmount.Sub(current))
	switch {
	case delta.IsPositive():
		allocated, err := s.allocate(ctx, allocateCreditRealizationsInput{Charge: in.Charge, Run: in.Run, AllocateAt: in.AllocateAt, AmountToAllocate: delta, CurrencyCalculator: in.CurrencyCalculator, Exact: in.ExactAllocation})
		if err != nil { return ReconcileCreditRealizationsResult{}, err }
		return ReconcileCreditRealizationsResult{Delta: delta, Realizations: allocated.Realizations}, nil
	case delta.IsNegative():
		// CreditsAllocated.Correct -> handler.OnCreditsOnlyUsageAccruedCorrection -> createRunCreditRealizations
	}
	return ReconcileCreditRealizationsResult{Delta: delta}, nil
}
```

<!-- archie:ai-end -->
