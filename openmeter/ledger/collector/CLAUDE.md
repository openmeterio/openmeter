# collector

<!-- archie:ai-start -->

> Implements FBO-to-accrued credit collection and lineage-aware correction (reversal) as a standalone service consumed by chargeadapter's flat-fee and usage-based handlers. Encapsulates advance shortfall issuance for CreditOnly mode and plan-then-execute merging for multi-step accrual corrections.

## Patterns

**Service interface with two operations only** — collector.Service exposes exactly CollectToAccrued and CorrectCollectedAccrued. Implementation delegates to private accrualCollector (collect.go) and accrualCorrector (correct.go) structs constructed in NewService(Config). (`func NewService(config Config) Service { return &service{collector: &accrualCollector{ledger: config.Ledger, deps: config.Dependencies}, corrector: &accrualCorrector{...}} }`)
**Advance shortfall issuance in collect (CreditOnly only)** — After resolving FBO→accrued templates, collect checks if settlement mode is CreditOnly and FBO didn't cover the full amount; if so it appends IssueCustomerReceivable + TransferCustomerFBOAdvanceToAccrued for the shortfall to the same group. CreditThenInvoice shortfall is never issued as advance. (`if shortfall := input.Amount.Sub(collectedFBOAmount); c.shouldAdvanceShortfall(input, shortfall) { advanceInputs, _ := c.resolveAdvanceInputs(ctx, input, shortfall); inputs = append(inputs, advanceInputs...) }`)
**Credit realization rows from FBO debits** — toCreditRealizations scans resolved inputs and creates one creditrealization.CreateAllocationInput per entry that debits AccountTypeCustomerFBO (negative amount on FBO sub-account). The transactionGroupID is shared across all realizations in the same group. (`for _, entry := range input.EntryInputs() { if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO { out = append(out, creditrealization.CreateAllocationInput{Amount: entry.Amount().Abs(), LedgerTransaction: ledgertransaction.GroupReference{TransactionGroupID: transactionGroupID}}) } }`)
**Plan-then-execute with per-transaction merging for corrections** — accrualCorrector.correct builds []plannedAction per correction item, merges overlapping corrections by transaction ID (to produce one aggregated amount per original transaction), then calls transactions.CorrectTransaction for each in correctionOrder. Changing execution order breaks the merging invariant. (`actions := planCorrection(...); mergedByTx := mergeByTransactionID(actions); for _, txID := range correctionOrder { c.ledger.CommitGroup(ctx, mergedByTx[txID]) }`)
**Lineage-aware segment dispatch in planCorrection** — When LineageSegmentsByRealization is present, planCorrection dispatches per segment.State: RealCredit → planSourceCorrectionActions, AdvanceUncovered → planAdvanceCorrection, AdvanceBackfilled → planBackfilledCorrection (which re-issues to FBO rather than immediately redirecting). (`switch segment.State { case LineageSegmentStateRealCredit: return plannedSourceCorrectionActions(source, amount, false) ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Public interface Service, Config, CollectToAccruedInput, CorrectCollectedAccruedInput, and NewService constructor. All external callers (chargeadapter flat-fee and usage-based handlers) use this file. | CollectToAccruedInput.Annotations is optional; if nil, ChargeAnnotations is synthesized from Namespace+ChargeID inside collect.go. CorrectCollectedAccruedInput.LineageSegmentsByRealization may be empty for legacy data — the corrector must handle this gracefully. |
| `collect.go` | accrualCollector — drives the FBO→accrued collection path and advance shortfall issuance. Returns creditrealization.CreateAllocationInputs (one per FBO debit). | shouldAdvanceShortfall only returns true for CreditOnlySettlementMode; never issue advances for CreditThenInvoice. |
| `correct.go` | accrualCorrector — reverses prior collections using lineage-aware planning. Most complex file; plan/execute split prevents double-counting when multiple corrections reference the same source transaction. | reissueBackfilledCredit re-issues purchased credits back to FBO without triggering a new backfill sweep — this is intentional; do not replace with a recursive advance correction. |
| `collection_fbo.go` | collectCustomerFBO — sorts and drains FBO sub-accounts in ascending creditPriority order to produce fboCollectionSource slices consumed by resolveCollectedInputs. | Priority ordering is ascending (lower priority number = drained first); changing cmp logic here breaks priority-ordered credit consumption. |

## Anti-Patterns

- Calling ledger.CommitGroup directly from charge handlers for FBO collection instead of routing through collector.Service
- Modifying the correction plan execution order (correctionOrder preserves insertion order; reordering breaks per-transaction merging)
- Assuming CorrectCollectedAccrued is idempotent — it must only be called once per correction batch
- Issuing advance shortfall for CreditThenInvoice mode (shouldAdvanceShortfall is CreditOnly-only)

## Decisions

- **Correction uses plan-then-execute with merged per-transaction amounts.** — Multiple realization corrections may reference the same original ledger transaction; merging before execution ensures one aggregated correction entry per original transaction, preventing over-correction.
- **Backfilled advance corrections re-issue to FBO rather than immediately redirecting to another uncovered advance.** — Prevents correction from triggering a cascading backfill pass that could unpredictably affect other outstanding advances.

## Example: Calling collector.Service from a charge handler to collect FBO credits

```
// In chargeadapter/flatfee.go or usagebased.go
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
