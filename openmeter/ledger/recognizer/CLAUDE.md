# recognizer

<!-- archie:ai-start -->

> Implements revenue recognition by transferring attributable accrued ledger balances to earnings accounts using lineage-aware segment splitting, exposing a single RecognizeEarnings operation wrapped in a Postgres transaction.

## Patterns

**Config struct with Validate() before NewService** — Config holds all dependencies (Ledger, ResolverDependencies, Lineage, TransactionManager) and validates they are non-nil before NewService proceeds. NewService returns (Service, error) — callers must handle the error. (`func NewService(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &service{...}, nil }`)
**transaction.Run wrapping entire recognition flow** — RecognizeEarnings wraps all DB reads and writes inside transaction.Run(ctx, s.transactionManager, ...) to ensure atomic segment splitting and ledger commit. (`return transaction.Run(ctx, s.transactionManager, func(ctx context.Context) (RecognizeEarningsResult, error) { ... })`)
**Collect eligible lineages then resolve recognition template** — RecognizeEarnings first loads all lineages for the customer+currency via s.lnge.LoadLineagesByCustomer, filters to recognizable segment states (RealCredit, AdvanceBackfilled), sums totalEligible, then resolves RecognizeEarningsFromAttributableAccruedTemplate against the actual ledger balance. (`lineages, _ := s.lnge.LoadLineagesByCustomer(ctx, ...); eligible := collectEligibleLineages(lineages); resolved, _ := transactions.ResolveTransactions(ctx, s.deps, scope, RecognizeEarningsFromAttributableAccruedTemplate{Amount: totalEligible, ...})`)
**Segment splitting with remainder creation** — For each eligible segment, consumed = min(segRemaining, remainingToAllocate). If consumed < segment amount, a remainder segment is created. Then an EarningsRecognized segment is created referencing the backing transaction group for reversal traceability. (`consumed := minDecimal(seg.Amount, remaining); if remainder := seg.Amount.Sub(consumed); remainder.IsPositive() { s.lnge.CreateSegment(ctx, lineage.CreateSegmentInput{Amount: remainder, State: seg.State, ...}) }`)
**NoopService for tests not exercising recognition** — NoopService implements Service with var _ Service = NoopService{} assertion and returns zero RecognizeEarningsResult. Used in tests that don't need recognition side-effects. (`func (NoopService) RecognizeEarnings(context.Context, RecognizeEarningsInput) (RecognizeEarningsResult, error) { return RecognizeEarningsResult{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, Config, RecognizeEarningsInput (with Validate()), RecognizeEarningsResult, and NewService constructor. | RecognizeEarningsInput.Validate() checks CustomerID, At (non-zero), and Currency — callers must provide all three. |
| `recognize.go` | RecognizeEarnings implementation: lineage loading, eligible segment collection, template resolution, segment splitting, and ledger commit. | recognizableSegmentStates map governs which segment states are eligible — adding a new LineageSegmentState that should be recognized requires updating this map. |
| `noop.go` | NoopService — returns zero results. Used in tests and potentially in credits-disabled paths that still instantiate the recognizer interface. | var _ Service = NoopService{} assertion ensures compile-time compliance when Service gains methods. |

## Anti-Patterns

- Calling RecognizeEarnings outside a Postgres transaction context (transaction.Run is the required wrapper — already enforced inside service, but must not be disabled)
- Adding new recognizable segment states without updating the recognizableSegmentStates map in recognize.go
- Constructing recognizer.Service without calling Config.Validate() first
- Skipping segment remainder creation when consumed < segment amount (leaves orphan balance in the lineage)

## Decisions

- **Recognition resolves against actual ledger accrued balance (not purely against lineage amounts).** — The real ledger accrued balance may differ from the sum of lineage amounts due to manual adjustments or advance shortfall issuances; resolving against the ledger guarantees the recognition entry matches actual accounting state.
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
