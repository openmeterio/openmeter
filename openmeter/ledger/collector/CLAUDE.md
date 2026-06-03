# collector

<!-- archie:ai-start -->

> Implements FBO-to-accrued credit collection and lineage-aware correction (reversal) as a standalone service consumed by chargeadapter's flat-fee and usage-based handlers. Encapsulates advance shortfall issuance for CreditOnly mode and plan-then-execute merging for multi-step accrual corrections.

## Patterns

**Service interface with two operations only** — collector.Service exposes exactly CollectToAccrued and CorrectCollectedAccrued; implementation delegates to private accrualCollector (collect.go) and accrualCorrector (correct.go) built in NewService(Config). (`func NewService(config Config) Service { return &service{collector: &accrualCollector{...}, corrector: &accrualCorrector{...}} }`)
**Advance shortfall issuance in collect (CreditOnly only)** — After resolving FBO->accrued templates, collect appends IssueCustomerReceivable + TransferCustomerFBOAdvanceToAccrued for any shortfall only when settlement mode is CreditOnly. CreditThenInvoice never issues advance. (`if shortfall := input.Amount.Sub(collectedFBOAmount); c.shouldAdvanceShortfall(input, shortfall) { inputs = append(inputs, advanceInputs...) }`)
**Credit realization rows from FBO debits** — toCreditRealizations emits one creditrealization.CreateAllocationInput per resolved entry that debits AccountTypeCustomerFBO, sharing the group's transactionGroupID. (`if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO { out = append(out, creditrealization.CreateAllocationInput{Amount: entry.Amount().Abs()}) }`)
**Plan-then-execute with per-transaction merging** — accrualCorrector.correct builds []plannedAction, merges overlapping corrections by transaction ID into one aggregated amount per original transaction, then CorrectTransaction per ID in correctionOrder. Reordering breaks the merge invariant. (`mergedByTx := mergeByTransactionID(actions); for _, txID := range correctionOrder { c.ledger.CommitGroup(ctx, mergedByTx[txID]) }`)
**Lineage-aware segment dispatch in planCorrection** — When LineageSegmentsByRealization is present, planCorrection dispatches per segment.State: RealCredit -> source correction, AdvanceUncovered -> advance correction, AdvanceBackfilled -> re-issue to FBO. (`switch segment.State { case LineageSegmentStateRealCredit: return plannedSourceCorrectionActions(source, amount, false) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Public Service interface, Config, CollectToAccruedInput, CorrectCollectedAccruedInput, NewService. Entry point for chargeadapter handlers. | CollectToAccruedInput.Annotations is optional (synthesized from Namespace+ChargeID if nil); CorrectCollectedAccruedInput.LineageSegmentsByRealization may be empty for legacy data and must be handled gracefully. |
| `collect.go` | accrualCollector — FBO->accrued collection and advance shortfall issuance; returns one CreateAllocationInput per FBO debit. | shouldAdvanceShortfall returns true only for CreditOnlySettlementMode; never issue advances for CreditThenInvoice. |
| `correct.go` | accrualCorrector — lineage-aware reversal with plan/execute split to avoid double-counting when multiple corrections reference the same source transaction. | reissueBackfilledCredit re-issues purchased credits to FBO without triggering a new backfill sweep — intentional; do not replace with a recursive advance correction. |
| `collection_fbo.go` | collectCustomerFBO drains FBO sub-accounts in ascending creditPriority order into fboCollectionSource slices. | Ordering is ascending (lower priority drained first); changing cmp logic breaks priority-ordered credit consumption and breakage release alignment. |

## Anti-Patterns

- Calling ledger.CommitGroup directly from charge handlers for FBO collection instead of routing through collector.Service
- Modifying the correction plan execution order (correctionOrder preserves insertion order; reordering breaks per-transaction merging)
- Assuming CorrectCollectedAccrued is idempotent — call it once per correction batch
- Issuing advance shortfall for CreditThenInvoice mode (shouldAdvanceShortfall is CreditOnly-only)

## Decisions

- **Correction uses plan-then-execute with merged per-transaction amounts.** — Multiple realization corrections may reference the same original ledger transaction; merging before execution yields one aggregated correction per original transaction, preventing over-correction.
- **Backfilled advance corrections re-issue to FBO rather than redirecting to another uncovered advance.** — Prevents correction from triggering a cascading backfill pass that could unpredictably affect other outstanding advances.

## Example: Calling collector.Service from a charge handler to collect FBO credits

```
realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{
    Namespace:      input.Charge.Namespace,
    ChargeID:       input.Charge.ID,
    CustomerID:     input.Charge.Intent.CustomerID,
    Annotations:    chargeAnnotationsForFlatFeeCharge(input.Charge),
    At:             input.Charge.Intent.InvoiceAt,
    Currency:       input.Charge.Intent.Currency,
    SettlementMode: input.Charge.Intent.SettlementMode,
    ServicePeriod:  input.ServicePeriod,
    Amount:         input.PreTaxTotalAmount,
})
if err != nil { return nil, err }
if len(realizations) == 0 { return nil, nil }
return realizations, nil
```

<!-- archie:ai-end -->
