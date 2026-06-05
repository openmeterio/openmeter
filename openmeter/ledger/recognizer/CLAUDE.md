# recognizer

<!-- archie:ai-start -->

> Revenue-recognition service: moves attributable accrued balance into earnings and advances lineage segments to earnings_recognized. It is the distinct recognition step (separate from accrual/acknowledgement) that books earnings against the ledger and transitions lineage state atomically.

## Patterns

**Single-method Service with Config/Validate constructor** — Service exposes only RecognizeEarnings; NewService(Config) validates required deps (Ledger, ResolverDependencies AccountService/AccountCatalog/BalanceQuerier, Lineage, TransactionManager) before building the struct. (`func NewService(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &service{...}, nil }`)
**Atomic recognition in transaction.Run** — RecognizeEarnings wraps load-lineages, resolve-template, CommitGroup, and lineage segment transitions in one transaction.Run so ledger and lineage state commit together. (`return transaction.Run(ctx, s.transactionManager, func(ctx context.Context) (RecognizeEarningsResult, error) { ... })`)
**Eligibility from lineage segment state** — collectEligibleLineages selects positive segments whose State is in recognizableSegmentStates (RealCredit, AdvanceBackfilled), sorted by lineage.ID for deterministic allocation. (`if recognizableSegmentStates[seg.State] && seg.Amount.IsPositive() { segments = append(segments, seg); amount = amount.Add(seg.Amount) }`)
**Resolve against actual accrued, recognize the real output** — Template RecognizeEarningsFromAttributableAccruedTemplate is resolved against ledger balance; the recognized amount is sumPositiveEntries(resolved), which may be less than eligible; zero output short-circuits. (`actualAmount := sumPositiveEntries(resolved); if !actualAmount.IsPositive() { return RecognizeEarningsResult{}, nil }`)
**Close-then-recreate segment transitions** — allocateRecognition closes each consumed source segment, recreates a remainder segment in the original state if partial, and creates an earnings_recognized segment carrying SourceState/SourceBackingTransactionGroupID for correction unwind. Keeps the active segment set non-overlapping. (`s.lnge.CloseSegment(ctx, seg.ID, now); s.lnge.CreateSegment(ctx, lineage.CreateSegmentInput{State: creditrealization.LineageSegmentStateEarningsRecognized, SourceState: &sourceState, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, Config/Validate, constructor, RecognizeEarningsInput/Result. | Recognition needs the full ResolverDependencies trio plus Lineage and TransactionManager; input requires non-zero At and a valid Currency. |
| `recognize.go` | RecognizeEarnings flow plus collectEligibleLineages and allocateRecognition. | Allocation order is deterministic by lineage.ID; segment time is At.Truncate(time.Microsecond); recognized amount allocated may be capped by actual ledger output, not eligible total. minDecimal/sumPositiveEntries are local helpers. |
| `noop.go` | NoopService returning empty RecognizeEarningsResult for tests not exercising recognition. | Keep it side-effect-free; var _ Service = NoopService{} must hold. |

## Anti-Patterns

- Committing the ledger recognition group without the lineage segment transitions in the same transaction.Run, desyncing earnings and lineage state.
- Recognizing the eligible total instead of the actual sumPositiveEntries(resolved) amount produced by the template against accrued balance.
- Mutating a lineage segment in place instead of close-then-recreate, producing overlapping active segments that break correction unwind.
- Dropping SourceState/SourceBackingTransactionGroupID on the earnings_recognized segment, making recognition non-reversible.
- Recognizing segments whose State is not in recognizableSegmentStates (only RealCredit / AdvanceBackfilled are eligible).

## Decisions

- **Recognition is gated on lineage segment state and resolved against actual accrued balance** — Only real or advance-backfilled credit may be recognized, and the ledger template (not the eligible total) determines how much accrued can actually be moved to earnings.
- **Segment transitions use close + recreate with source metadata** — Keeps the active segment set non-overlapping and records prior state so a later correction can unwind recognition back to the original segment state.

## Example: Recognizing earnings for eligible accrued and transitioning a segment

```
resolved, _ := transactions.ResolveTransactions(ctx, s.deps, transactions.ResolutionScope{CustomerID: in.CustomerID, Namespace: in.CustomerID.Namespace}, transactions.RecognizeEarningsFromAttributableAccruedTemplate{At: in.At, Amount: totalEligible, Currency: in.Currency})
actualAmount := sumPositiveEntries(resolved)
group, _ := s.ledger.CommitGroup(ctx, transactions.GroupInputs(in.CustomerID.Namespace, nil, resolved...))
sourceState := seg.State
s.lnge.CloseSegment(ctx, seg.ID, now)
s.lnge.CreateSegment(ctx, lineage.CreateSegmentInput{LineageID: seg.LineageID, Amount: consumed, State: creditrealization.LineageSegmentStateEarningsRecognized, BackingTransactionGroupID: &groupID, SourceState: &sourceState, SourceBackingTransactionGroupID: seg.BackingTransactionGroupID})
```

<!-- archie:ai-end -->
