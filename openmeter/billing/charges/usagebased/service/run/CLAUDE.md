# run

<!-- archie:ai-start -->

> Owns the mechanics of usage-based realization runs — rating, run persistence, credit allocation/correction, lineage writes, invoice usage booking, and payment authorization/settlement. Executes run operations only; it never makes state-machine decisions (those live in the usagebased charge service above).

## Patterns

**Config-struct constructor with Validate()** — New(Config) validates all four required dependencies (Adapter, Rater, Handler, Lineage) via Config.Validate() (errors.Join over []error) before returning *Service. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &Service{...}, nil }`)
**Input structs validate first** — Every exported method calls in.Validate() as its first action; input structs implement Validate() and recursively validate sub-fields with wrapped fmt.Errorf messages. (`func (s *Service) ReconcileCredits(ctx context.Context, in ReconcileCreditRealizationsInput) (...) { if err := in.Validate(); err != nil { return ..., err } }`)
**CreditAllocationMode enum for credit semantics** — Three-valued enum CreditAllocationNone/Exact/Available controls whether credits are allocated and whether exact-match is required; its Validate() returns a GenericValidationError on invalid values. (`const ( CreditAllocationNone CreditAllocationMode = "none"; CreditAllocationExact = "exact"; CreditAllocationAvailable = "available" )`)
**Atomic credit-realization + lineage writes** — createRunCreditRealizations always pairs adapter.CreateRunCreditRealization with the subsequent lineage CreateInitialLineages and PersistCorrectionLineageSegments calls — these must never be split. (`realizations, err := s.createRunCreditRealizations(ctx, in.Charge, in.Run.ID, corrections)`)
**Handler callbacks for all ledger side-effects** — All ledger writes (credit allocation, correction, payment auth/settlement, invoice usage accrual) go through usagebased.Handler callbacks; never call ledger adapters directly. (`s.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{...})`)
**Round before compare or persist** — Decimals must be rounded via CurrencyCalculator.RoundToPrecision before negative-total checks, delta comparisons, and UpdateRealizationRun calls; never compare unrounded alpacadecimal.Decimal values. (`delta := in.CurrencyCalculator.RoundToPrecision(in.TargetAmount.Sub(currentAmount))`)
**NoFiatTransactionRequired short-circuit** — BookInvoicedPaymentAuthorized and SettleInvoicedPayment short-circuit with a no-op result when in.Run.NoFiatTransactionRequired is true, skipping handler and adapter calls. (`if in.Run.NoFiatTransactionRequired { return BookInvoicedPaymentAuthorizedResult{Run: in.Run}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config, New constructor, and CreditAllocationMode enum. | No state-machine logic ever belongs here — this package executes runs, it does not decide which state to enter. |
| `create.go` | CreateRatedRun: full run creation — rate, validate totals, persist run + detailed lines, optionally allocate credits, update run totals. | runTotals must be rounded before the negative-total check and before UpdateRealizationRun; ensureDetailedLinesLoadedForRating must run before rating to satisfy PriorRuns validation. |
| `correct.go` | ReconcileCredits (delta allocation/correction to a TargetAmount) and CorrectAllCredits (zero out run credits). | TargetAmount and currentAmount must both be rounded before computing delta — comparing unrounded decimals produces incorrect billing corrections. |
| `credits.go` | Private allocate() helper and createRunCreditRealizations() pairing adapter credit creation with lineage writes. | Never call adapter.CreateRunCreditRealization without the two following lineage calls — they are a single atomic business operation. |
| `invoice.go` | BookAccruedInvoiceUsage: books invoice usage accrual via handler then persists AccruedUsage; short-circuits on NoFiatTransactionRequired. | An empty TransactionGroupID returned from the handler is an error — enforce this before calling adapter.CreateRunInvoicedUsage. |
| `payment.go` | BookInvoicedPaymentAuthorized and SettleInvoicedPayment: authorize/settle via handler then persist via adapter. | State guards in Validate() (payment already authorized, mismatched line ID, already settled) must be preserved for any new payment methods. |
| `payment_test.go` | Unit tests for the payment input Validate() methods covering all rejection paths. | Test helpers (newUsageBasedCharge, newUsageBasedRun) construct values directly using internal types — do not extract them to testutils. |

## Anti-Patterns

- Making state-machine decisions (status transitions, trigger selection) here — those belong in the usagebased charge service layer above.
- Calling ledger adapters directly — all ledger side-effects must flow through usagebased.Handler callbacks.
- Splitting createRunCreditRealizations from its lineage calls — credit creation and lineage writes must stay atomic.
- Comparing alpacadecimal.Decimal values before rounding with CurrencyCalculator — rounding order is required for billing correctness.
- Assigning charge.Intent.TaxConfig pointer directly into DetailedLine — always Clone() to prevent mutation across run boundaries.

## Decisions

- **Service owns run mechanics but not state-machine decisions** — Separating 'how to execute a run' from 'when to execute a run' keeps this package testable without a full state machine.
- **Handler interface for all ledger side-effects** — Ledger calls are externally provided via usagebased.Handler so this package compiles and tests without a real ledger; one seam per operation type.
- **CreditAllocationMode enum instead of boolean flags** — Three distinct allocation semantics (none, exact, available) cannot be expressed with a boolean; an enum with Validate() catches invalid values at construction.

## Example: Construct the run service and create a rated run

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"

runSvc, err := run.New(run.Config{
	Adapter: usagebasedAdapter,
	Rater:   ratingService,
	Handler: usagebasedHandler,
	Lineage: lineageService,
})
if err != nil { return err }

result, err := runSvc.CreateRatedRun(ctx, run.CreateRatedRunInput{
	Charge:           charge,
	CustomerOverride: customerOverride,
	CreditAllocation: run.CreditAllocationExact,
})
```

<!-- archie:ai-end -->
