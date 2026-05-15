# realizations

<!-- archie:ai-start -->

> Owns flat-fee credit allocation and correction mechanics — persisting credit realization records and their lineage segments. It must not make state-machine decisions; all charge lifecycle transitions happen in the parent charges.Service layer.

## Patterns

**Config struct with Validate() before construction** — All dependencies declared in Config; Config.Validate() collects nil-check errors via errors.Join before New() returns. Never construct Service directly. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Input structs carry Validate() called at method entry** — Each method input (AllocateCreditsOnlyInput, CorrectAllCreditRealizationsInput, AccrueInvoiceUsageInput, StartCreditThenInvoiceRunInput) has Validate() that checks zero-values, negative amounts, and cross-field invariants. Call Validate() at the top of every service method before any adapter call. (`func (s *Service) AllocateCreditsOnly(ctx context.Context, in AllocateCreditsOnlyInput) (AllocateCreditsOnlyResult, error) { if err := in.Validate(); err != nil { return AllocateCreditsOnlyResult{}, err } ... }`)
**Currency rounding before validation and sum assertion** — RoundToPrecision is called on monetary amounts before Validate() and before sum equality checks to avoid floating-point false negatives. Pattern: round first, then validate, then assert sum equality. (`in.Amount = in.CurrencyCalculator.RoundToPrecision(in.Amount); allocated := in.CurrencyCalculator.RoundToPrecision(creditAllocations.Sum()); if !allocated.Equal(in.Amount) { return ..., models.NewGenericValidationError(...) }`)
**createCreditAllocations always triples: adapter write + CreateInitialLineages + PersistCorrectionLineageSegments** — The private createCreditAllocations method always calls adapter.CreateCreditAllocations, then lineage.CreateInitialLineages, then lineage.PersistCorrectionLineageSegments — all three within the caller's transaction. Skipping either lineage step breaks correction lookups in CorrectAllCredits. (`realizations, err := s.adapter.CreateCreditAllocations(ctx, runID, creditAllocations); s.lineage.CreateInitialLineages(ctx, ...); s.lineage.PersistCorrectionLineageSegments(ctx, ...)`)
**Handler delegates allocation-splitting decisions** — Service does not compute how credits are split across ledger accounts. It calls flatfee.Handler methods (OnCreditsOnlyUsageAccrued, OnCreditsOnlyUsageAccruedCorrection, OnAssignedToInvoice, OnInvoiceUsageAccrued) and only validates sum equality afterward. (`creditAllocations, err := s.handler.OnCreditsOnlyUsageAccrued(ctx, input); allocated := currencyCalculator.RoundToPrecision(creditAllocations.Sum()); if !allocated.Equal(in.Amount) { return ..., models.NewGenericValidationError(...) }`)
**Multi-step writes wrapped in transaction.Run** — Methods that combine adapter writes (e.g. CreateCurrentRun + CreateCreditAllocations + UpdateRealizationRun + UpsertDetailedLines) wrap all steps in transaction.Run(ctx, s.adapter, func(ctx) ...) to guarantee atomicity. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (StartCreditThenInvoiceRunResult, error) { ... s.adapter.CreateCurrentRun(...); s.createCreditAllocations(...); s.adapter.UpdateRealizationRun(...) })`)
**sum-mismatch returns models.NewGenericValidationError, not an internal error** — When allocated sum does not equal expected amount after rounding, return models.NewGenericValidationError — not fmt.Errorf — so the GenericErrorEncoder maps it to HTTP 400. (`if !allocated.Equal(in.Amount) { return AllocateCreditsOnlyResult{}, models.NewGenericValidationError(fmt.Errorf("credit allocations do not match total [charge_id=%s ...]", ...)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct, Config, Config.Validate(), New(), and the private createCreditAllocations helper that enforces the triple-write invariant (adapter + initial lineage + correction lineage segments). | Adding new exported methods here without following the Config.Validate + Input.Validate + transaction.Run pattern. Also watch for missing lineage.PersistCorrectionLineageSegments calls after adapter.CreateCreditAllocations. |
| `creditsonly.go` | Implements AllocateCreditsOnly: rounds amount, delegates split to handler.OnCreditsOnlyUsageAccrued, asserts sum equality, persists realizations and updates run totals inside transaction.Run. | Skipping the sum equality assertion or performing arithmetic without RoundToPrecision. |
| `correct.go` | Implements CorrectAllCredits: loads active lineage segments, delegates corrections to handler.OnCreditsOnlyUsageAccruedCorrection via creditrealization.CorrectAll, and persists correction realizations. | Calling createCreditAllocations without first loading lineage segments via lineage.LoadActiveSegmentsByRealizationID. |
| `credittheninvoice.go` | Implements StartCreditThenInvoiceRun: creates realization run, allocates credits via handler.OnAssignedToInvoice, generates and merges detailed billing lines via ratingService, persists everything in one transaction. | Cross-field validation in StartCreditThenInvoiceRunInput.Validate() checks that line.ChargeID == charge.ID and line.InvoiceID == invoice.ID — new callers must satisfy both before calling. |
| `invoiceaccrued.go` | Implements AccrueInvoiceUsage: calls handler.OnInvoiceUsageAccrued to post a ledger transaction, persists invoiced usage record, then marks the run as immutable. | AccrueInvoiceUsageInput.Validate() rejects inputs where currentRun.AccruedUsage != nil (already accrued) or where run IDs don't match the provided line/invoice IDs. |

## Anti-Patterns

- Making state-machine decisions (charge status transitions) inside this package — all lifecycle transitions belong in the parent charges.Service layer
- Calling adapter.CreateCreditAllocations without following with both lineage.CreateInitialLineages and lineage.PersistCorrectionLineageSegments in the same transaction
- Performing currency arithmetic without rounding via CurrencyCalculator.RoundToPrecision before validation or sum comparison
- Returning fmt.Errorf for allocation sum mismatches instead of models.NewGenericValidationError (breaks HTTP 400 mapping)
- Skipping Config.Validate() or Input.Validate() before adapter calls

## Decisions

- **Handler delegates allocation-splitting; Service only validates sum and persists** — Different flat-fee billing strategies (immediate full allocation, pro-rated, credit-then-invoice) vary the split logic; keeping split decisions in flatfee.Handler lets realization persistence remain strategy-agnostic.
- **Lineage persistence is mandatory and co-located in the private createCreditAllocations helper** — Realization records without lineage segments break correction lookups in CorrectAllCredits; coupling adapter write + lineage in one method makes it structurally impossible to forget lineage steps.
- **Config+Validate pattern instead of variadic functional options** — Consistent with the broader billing/charges constructor convention; surfaces missing-dependency errors at startup (New() call) rather than at first method invocation.

## Example: Allocate credits for a flat-fee charge, persist realizations and lineage, assert sum equality

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
)

svc, err := realizations.New(realizations.Config{
	Adapter:       adapter,
	Handler:       handler,
	Lineage:       lineageSvc,
	RatingService: ratingSvc,
})
if err != nil {
	return err
}

result, err := svc.AllocateCreditsOnly(ctx, realizations.AllocateCreditsOnlyInput{
// ...
```

<!-- archie:ai-end -->
