# run

<!-- archie:ai-start -->

> Owns the mechanics of usage-based realization runs: creating rated runs, persisting detailed lines, allocating/correcting credits, booking invoice usage, and recording payments. Deliberately excludes state-machine decisions — it executes run operations but never decides which state to enter.

## Patterns

**Config-struct constructor with Validate()** — New(Config) validates all four required dependencies (Adapter, Rater, Handler, Lineage) before returning *Service. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Every exported method validates its input struct first** — All exported methods call in.Validate() as their first action. Input structs implement Validate() and validate sub-fields recursively with wrapped error messages. (`func (s *Service) CreateRatedRun(ctx context.Context, in CreateRatedRunInput) (...) { if err := in.Validate(); err != nil { return ..., err } }`)
**CreditAllocationMode enum controls credit allocation behaviour** — CreditAllocationNone/Exact/Available passed into CreateRatedRun determines whether credits are allocated, and whether exact-match is required. (`CreditAllocation: CreditAllocationExact // allocates exact run total; error if ledger cannot satisfy`)
**Lineage created immediately after credit realizations** — createRunCreditRealizations always calls lineage.CreateInitialLineages and lineage.PersistCorrectionLineageSegments right after adapter.CreateRunCreditRealization — never split these calls. (`s.adapter.CreateRunCreditRealization(...); s.lineage.CreateInitialLineages(...); s.lineage.PersistCorrectionLineageSegments(...)`)
**Handler callbacks for ledger side-effects** — Ledger writes (credit allocation, payment authorization, settlement, invoice usage accrual) go through usagebased.Handler callbacks (OnCreditsOnlyUsageAccrued, OnInvoiceUsageAccrued, OnPaymentAuthorized, OnPaymentSettled) — never call ledger directly. (`ledgerRef, err := s.handler.OnInvoiceUsageAccrued(ctx, usagebased.OnInvoiceUsageAccruedInput{...})`)
**ensureDetailedLinesLoadedForRating before creating a new run** — CreateRatedRun calls ensureDetailedLinesLoadedForRating to lazy-fetch prior run detailed lines from the adapter if any run lacks them — required by the rating package's PriorRuns validation. (`chargeWithDetailedLines, err := s.ensureDetailedLinesLoadedForRating(ctx, in.Charge)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct, Config, New constructor, and CreditAllocationMode enum. | Service intentionally has no state-machine logic — never add status transition decisions here. |
| `create.go` | CreateRatedRun: full run creation flow — rate, validate totals, persist run, persist detailed lines, optionally allocate credits. | runTotals must be rounded with CurrencyCalculator before negative-total check and before UpdateRealizationRun. |
| `correct.go` | ReconcileCredits (delta allocation/correction) and CorrectAllCredits for zeroing out run credits. Uses lineage.LoadActiveSegmentsByRealizationID before generating corrections. | ReconcileCredits rounds TargetAmount and currentAmount before computing delta — never compare unrounded decimals. |
| `credits.go` | Private allocate() helper and createRunCreditRealizations() that pairs credit creation with lineage writes. | Never call adapter.CreateRunCreditRealization without the two subsequent lineage calls — they must be atomic from a business logic standpoint. |
| `invoice.go` | BookAccruedInvoiceUsage: books the invoice usage accrual via handler then persists AccruedUsage. Short-circuits on zero-total lines. | Empty TransactionGroupID from handler is an error — enforce this check before persisting. |
| `payment.go` | BookInvoicedPaymentAuthorized and SettleInvoicedPayment: authorize/settle payment via handler then persist via adapter. | State guards in Validate() (payment already authorized, mismatched line ID, already settled) must be preserved in any new payment methods. |
| `detailedline.go` | mapRatingResultToRunDetailedLines (rating->domain mapping) and PersistRunDetailedLines (upsert to adapter). | TaxConfig must be cloned via cfg.Clone() — never assign the charge pointer directly or mutations will bleed across runs. |

## Anti-Patterns

- Making state-machine decisions (status transitions, trigger selection) inside this package — those belong in the usagebased charge service layer above.
- Calling ledger adapters directly — all ledger side-effects must flow through usagebased.Handler callbacks.
- Splitting createRunCreditRealizations from its lineage calls — CreateInitialLineages and PersistCorrectionLineageSegments must always follow credit creation.
- Comparing alpacadecimal.Decimal values before rounding with CurrencyCalculator — rounding order matters for billing correctness.
- Assigning charge.Intent.TaxConfig pointer directly into DetailedLine — always clone with cfg.Clone() to prevent mutation across run boundaries.

## Decisions

- **Service owns run mechanics but not state-machine decisions** — Separating the 'how to execute a run' (rating, persistence, credit math) from 'when to execute a run' (state transitions) keeps this package testable without a full state machine and prevents accidental coupling of lifecycle logic to run mechanics.
- **Handler interface for all ledger side-effects** — Ledger calls (credit allocation, payment, invoice accrual) are externally provided via usagebased.Handler so this package compiles and tests without a real ledger implementation, and the ledger integration point is a single seam per operation type.
- **CreditAllocationMode enum instead of boolean flags** — Three distinct allocation semantics (none, exact, available) cannot be expressed cleanly with a boolean; an enum with Validate() catches invalid values at construction time and self-documents caller intent.

## Example: Create a final realization run with exact credit allocation

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
    usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
)

runSvc, err := run.New(run.Config{
    Adapter: usagebasedAdapter,
    Rater:   ratingService,
    Handler: usagebasedHandler,
    Lineage: lineageService,
})
if err != nil { return err }

result, err := runSvc.CreateRatedRun(ctx, run.CreateRatedRunInput{
    Charge:                  charge,
// ...
```

<!-- archie:ai-end -->
