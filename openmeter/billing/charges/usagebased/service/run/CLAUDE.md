# run

<!-- archie:ai-start -->

> Owns the mechanics of usage-based realization runs — rating, run persistence, credit allocation/correction, lineage writes, invoice usage booking, and payment authorization/settlement. Never makes state-machine decisions; executes run operations only.

## Patterns

**Config-struct constructor with Validate()** — New(Config) validates all four required dependencies (Adapter, Rater, Handler, Lineage) before returning *Service. Config.Validate() uses errors.Join over a []error slice. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } return &Service{...}, nil }`)
**Input structs validate first** — Every exported method calls in.Validate() as its first action. Input structs implement Validate() and validate sub-fields recursively with wrapped fmt.Errorf messages. (`func (s *Service) CreateRatedRun(ctx context.Context, in CreateRatedRunInput) (...) { if err := in.Validate(); err != nil { return ..., err } ... }`)
**CreditAllocationMode enum for credit semantics** — Three-valued enum CreditAllocationNone/Exact/Available controls whether credits are allocated and whether exact-match is required. Its own Validate() catches invalid values. (`CreditAllocation: CreditAllocationExact // allocates exact run total; error if ledger cannot satisfy`)
**Atomic credit-realization + lineage writes** — createRunCreditRealizations always calls lineage.CreateInitialLineages then lineage.PersistCorrectionLineageSegments immediately after adapter.CreateRunCreditRealization. These three calls must never be split. (`s.adapter.CreateRunCreditRealization(...); s.lineage.CreateInitialLineages(...); s.lineage.PersistCorrectionLineageSegments(...)`)
**Handler callbacks for all ledger side-effects** — Ledger writes (credit allocation, payment authorization, settlement, invoice usage accrual) go through usagebased.Handler callbacks. Never call ledger adapters directly from this package. (`ledgerRef, err := s.handler.OnInvoiceUsageAccrued(ctx, usagebased.OnInvoiceUsageAccruedInput{...})`)
**Round before compare or persist** — Decimals must be rounded via CurrencyCalculator.RoundToPrecision before negative-total checks, delta comparisons, and UpdateRealizationRun calls. Never compare unrounded alpacadecimal.Decimal values. (`runTotals := ratingResult.Totals.RoundToPrecision(in.CurrencyCalculator); if runTotals.Total.IsNegative() { ... }`)
**NoFiatTransactionRequired short-circuit** — BookInvoicedPaymentAuthorized and SettleInvoicedPayment both short-circuit with a no-op result when in.Run.NoFiatTransactionRequired is true, skipping handler and adapter calls. (`if in.Run.NoFiatTransactionRequired { return BookInvoicedPaymentAuthorizedResult{Run: in.Run}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct, Config, New constructor, and CreditAllocationMode enum. | No state-machine logic ever belongs here — this package executes runs, it does not decide which state to enter. |
| `create.go` | CreateRatedRun: full run creation — rate, validate totals, persist run+detailed lines, optionally allocate credits, update run totals. | runTotals must be rounded before the negative-total check and before UpdateRealizationRun. ensureDetailedLinesLoadedForRating must be called before rating to satisfy PriorRuns validation. |
| `correct.go` | ReconcileCredits (delta allocation/correction) and CorrectAllCredits (zero out run credits). | TargetAmount and currentAmount must both be rounded before computing delta — comparing unrounded decimals produces incorrect billing corrections. |
| `credits.go` | Private allocate() helper and createRunCreditRealizations() that pairs adapter credit creation with lineage writes. | Never call adapter.CreateRunCreditRealization without the two subsequent lineage calls — they must be treated as a single atomic business operation. |
| `invoice.go` | BookAccruedInvoiceUsage: books invoice usage accrual via handler then persists AccruedUsage. Short-circuits on NoFiatTransactionRequired. | Empty TransactionGroupID returned from handler is an error — enforce this check before calling adapter.CreateRunInvoicedUsage. |
| `payment.go` | BookInvoicedPaymentAuthorized and SettleInvoicedPayment: authorize/settle payment via handler then persist via adapter. | State guards in Validate() (payment already authorized, mismatched line ID, already settled) must be preserved for any new payment methods. |
| `payment_test.go` | Unit tests for BookInvoicedPaymentAuthorizedInput.Validate() and SettleInvoicedPaymentInput.Validate() covering all rejection paths. | Test helpers (newUsageBasedCharge, newUsageBasedRun) construct values directly — do not extract them to testutils; they depend on internal types. |

## Anti-Patterns

- Making state-machine decisions (status transitions, trigger selection) inside this package — those belong in the usagebased charge service layer above.
- Calling ledger adapters directly — all ledger side-effects must flow through usagebased.Handler callbacks.
- Splitting createRunCreditRealizations from its lineage calls — CreateInitialLineages and PersistCorrectionLineageSegments must always follow credit creation atomically.
- Comparing alpacadecimal.Decimal values before rounding with CurrencyCalculator — rounding order is required for billing correctness.
- Assigning charge.Intent.TaxConfig pointer directly into DetailedLine — always clone with cfg.Clone() to prevent mutation across run boundaries.

## Decisions

- **Service owns run mechanics but not state-machine decisions** — Separating 'how to execute a run' (rating, persistence, credit math) from 'when to execute a run' (state transitions) keeps this package testable without a full state machine.
- **Handler interface for all ledger side-effects** — Ledger calls are externally provided via usagebased.Handler so this package compiles and tests without a real ledger implementation; the integration point is a single seam per operation type.
- **CreditAllocationMode enum instead of boolean flags** — Three distinct allocation semantics (none, exact, available) cannot be expressed cleanly with a boolean; an enum with Validate() catches invalid values at construction time.

## Example: Create a final realization run with exact credit allocation

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
)

runSvc, err := run.New(run.Config{
    Adapter: usagebasedAdapter,
    Rater:   ratingService,
    Handler: usagebasedHandler,
    Lineage: lineageService,
})
if err != nil { return err }

result, err := runSvc.CreateRatedRun(ctx, run.CreateRatedRunInput{
    Charge:             charge,
    CustomerOverride:   customerOverride,
// ...
```

<!-- archie:ai-end -->
