# recognizer

<!-- archie:ai-start -->

> Implements revenue recognition by transferring attributable accrued ledger balances to earnings accounts using lineage-aware segment splitting, exposing a single RecognizeEarnings operation wrapped in a Postgres transaction.

## Patterns

**Config struct with Validate() before NewService** — Config holds all deps (Ledger, ResolverDependencies, Lineage, TransactionManager) and validates non-nil before NewService; NewService returns (Service, error). (`func NewService(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &service{...}, nil }`)
**transaction.Run wrapping recognition flow** — RecognizeEarnings wraps all DB reads/writes inside transaction.Run for atomic segment splitting plus ledger commit. (`return transaction.Run(ctx, s.transactionManager, func(ctx context.Context) (RecognizeEarningsResult, error) { ... })`)
**Collect eligible lineages then resolve recognition template** — Loads lineages via LoadLineagesByCustomer, filters to recognizable states (RealCredit, AdvanceBackfilled), sums totalEligible, then resolves RecognizeEarningsFromAttributableAccruedTemplate against the actual ledger balance. (`eligible := collectEligibleLineages(lineages); resolved, _ := transactions.ResolveTransactions(ctx, s.deps, scope, RecognizeEarningsFromAttributableAccruedTemplate{Amount: totalEligible})`)
**Segment splitting with remainder creation** — For each eligible segment consumed=min(segRemaining, remaining); if consumed<segment amount a remainder segment is created, then an EarningsRecognized segment referencing the backing group for reversal traceability. (`consumed := minDecimal(seg.Amount, remaining); if rem := seg.Amount.Sub(consumed); rem.IsPositive() { s.lnge.CreateSegment(ctx, lineage.CreateSegmentInput{Amount: rem, State: seg.State}) }`)
**NoopService for tests not exercising recognition** — NoopService implements Service (var _ Service = NoopService{}) and returns a zero RecognizeEarningsResult. (`func (NoopService) RecognizeEarnings(context.Context, RecognizeEarningsInput) (RecognizeEarningsResult, error) { return RecognizeEarningsResult{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, Config, RecognizeEarningsInput (with Validate()), RecognizeEarningsResult, NewService. | RecognizeEarningsInput.Validate() requires CustomerID, non-zero At, and Currency — all three are mandatory. |
| `recognize.go` | RecognizeEarnings impl: lineage loading, eligible segment collection, template resolution, segment splitting, ledger commit. | recognizableSegmentStates map (RealCredit, AdvanceBackfilled) governs eligibility — adding a new recognizable LineageSegmentState requires updating this map. |
| `noop.go` | NoopService returning zero results for tests and credits-disabled paths. | var _ Service = NoopService{} assertion ensures compile-time compliance as Service evolves. |

## Anti-Patterns

- Calling RecognizeEarnings outside a Postgres transaction (transaction.Run is the required wrapper; must not be disabled)
- Adding recognizable segment states without updating recognizableSegmentStates in recognize.go
- Constructing recognizer.Service without calling Config.Validate() first
- Skipping segment remainder creation when consumed < segment amount (leaves orphan balance in the lineage)

## Decisions

- **Recognition resolves against the actual ledger accrued balance, not purely against lineage amounts.** — The ledger balance may differ from summed lineage amounts due to manual adjustments or advance shortfall issuance; resolving against the ledger keeps the entry matched to real accounting state.
- **Segment splitting stores SourceState and SourceBackingTransactionGroupID on EarningsRecognized segments.** — Enables future correction to unwind recognition back to the prior state by restoring the original segment state and its backing group reference.

## Example: Calling RecognizeEarnings with full input

```
import (
    "github.com/openmeterio/openmeter/openmeter/ledger/recognizer"
    "github.com/openmeterio/openmeter/openmeter/customer"
    "github.com/openmeterio/openmeter/pkg/currencyx"
)

result, err := recognizerSvc.RecognizeEarnings(ctx, recognizer.RecognizeEarningsInput{
    CustomerID: customer.CustomerID{Namespace: ns, ID: customerID},
    At:         time.Now().UTC(),
    Currency:   currencyx.Code("USD"),
})
if err != nil { return fmt.Errorf("recognize earnings: %w", err) }
if result.RecognizedAmount.IsZero() { /* nothing to recognize */ }
```

<!-- archie:ai-end -->
