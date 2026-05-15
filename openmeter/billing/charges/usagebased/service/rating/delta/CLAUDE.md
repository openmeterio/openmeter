# delta

<!-- archie:ai-start -->

> Implements the production delta rating engine for usage-based charges: rates the current cumulative meter snapshot via billingrating.Service, subtracts all already-billed DetailedLines, and books the remaining delta on the current run's service period. Every output line is stamped to the current period — corrections are not attributed back to their originating period.

## Patterns

**Cumulative-snapshot subtraction** — Engine.Rate always calls ratingService.GenerateDetailedLines on the full cumulative quantity, then calls subtract.SubtractRatedRunDetails to remove what was already billed. Never compute deltas at the meter-query level. (`billingDetailedLines, err := e.ratingService.GenerateDetailedLines(usagebased.RateableIntent{...}); remainingDetailedLines, err := subtract.SubtractRatedRunDetails(currentDetailedLines, alreadyBilledDetailedLines, uniqueReferenceIDGenerator{})`)
**Credit stripping before subtraction** — Strip CreditsApplied and CreditsTotal from alreadyBilledDetailedLines before passing to SubtractRatedRunDetails so credit-allocation changes do not appear as usage or pricing changes. (`line.CreditsApplied = nil; line.Totals.CreditsTotal = alpacadecimal.Zero; line.Totals.Total = line.Totals.CalculateTotal()`)
**Minimum commitment deferred to final period** — Pass billingrating.WithMinimumCommitmentIgnored() to GenerateDetailedLines whenever CurrentPeriod.ServicePeriod.To is before Intent.ServicePeriod.To. Minimum commitment is only applied on the final service-period snapshot. (`if in.CurrentPeriod.ServicePeriod.To.Before(in.Intent.ServicePeriod.To) { opts = append(opts, billingrating.WithMinimumCommitmentIgnored()) }`)
**All delta lines stamped to current period** — After subtraction, set every remaining line's ServicePeriod to CurrentPeriod.ServicePeriod and clear CorrectsRunID. Delta rating does not preserve the originating period of corrections. (`remainingDetailedLines[idx].ServicePeriod = in.CurrentPeriod.ServicePeriod; remainingDetailedLines[idx].CorrectsRunID = nil`)
**Dense index assignment after Sort** — Call remainingDetailedLines.Sort() before assigning Index values. Indexes are part of the detailed-line persistence contract and must be assigned after sorting. (`remainingDetailedLines.Sort(); for idx := range remainingDetailedLines { index := idx; remainingDetailedLines[idx].Index = &index }`)
**Duplicate ChildUniqueReferenceID validation** — After index assignment, validate that all ChildUniqueReferenceID values in the output are unique. Return an error if duplicates exist. (`childUniqueReferenceIDs := lo.GroupBy(remainingDetailedLines, ...); if len(lines) > 1 { return Result{}, fmt.Errorf("duplicate child unique reference id: %s", id) }`)
**PreviousOnlyReversal requires a persisted line ID** — uniqueReferenceIDGenerator.PreviousOnlyReversal errors if line.ID is empty because the correction child reference must be stable and deterministic. Always ensure already-billed lines carry their persisted ID. (`func (uniqueReferenceIDGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) { if line.ID == "" { return "", fmt.Errorf("detailed line id is required") }; return fmt.Sprintf("%s#correction:detailed_line_id=%s", line.PricerReferenceID, line.ID), nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | The single public entry point Engine.Rate. Owns the full delta-rating pipeline: validate input, call billing rating, convert to usagebased.DetailedLines, strip credits, subtract, stamp periods, sort, assign indexes, validate uniqueness. | Credits must be stripped before SubtractRatedRunDetails is called; omitting this makes credit changes appear as usage changes. WithCreditsMutatorDisabled() must also be passed to GenerateDetailedLines. |
| `uniquereferenceid.go` | Implements subtract.UniqueReferenceIDGenerator for the delta engine. CurrentOnly and MatchedDelta pass through the generated ChildUniqueReferenceID unchanged; PreviousOnlyReversal generates a deterministic correction reference using the persisted line ID. | PreviousOnlyReversal panics if line.ID is empty. The correction format is '<PricerReferenceID>#correction:detailed_line_id=<ID>' — do not change this format without updating downstream parsers. |
| `base_test.go` | Defines deltaRatingTestCase / deltaRatingPhase table-driven test harness. runDeltaRatingTestCase simulates multi-phase billing by passing booked lines from earlier phases as AlreadyBilledDetailedLines to later phases. | detailedLinesBookedForDeltaTest assigns synthetic IDs ('phase-N-line-M') so PreviousOnlyReversal can generate stable correction references. Tests that omit line IDs will fail when reversals are expected. |
| `engine_test.go` | Unit tests for Engine.Rate that use a stubRatingService to control billing-rating output precisely. Covers credit stripping, partial-run minimum-commitment suppression, subtraction, and correction reference generation. | stubRatingService.lastOpts lets tests assert that IgnoreMinimumCommitment and DisableCreditsMutator were set correctly — check these assertions when modifying option logic. |

## Anti-Patterns

- Do not pass credits-inclusive AlreadyBilledDetailedLines to SubtractRatedRunDetails without first zeroing CreditsApplied and CreditsTotal — credit changes will appear as spurious usage deltas.
- Do not assign Index values before calling Sort — indexes are persistence keys and must reflect final sort order.
- Do not set CorrectsRunID on output lines — delta rating intentionally books all corrections on the current run period; CorrectsRunID is for period-preserving rating only.
- Do not apply minimum commitment on partial runs — call WithMinimumCommitmentIgnored() whenever CurrentPeriod.ServicePeriod.To < Intent.ServicePeriod.To.
- Do not construct Engine without a non-nil billingrating.Service — billing rating is the source of the cumulative line list that subtraction operates on.

## Decisions

- **Delta engine books all corrections on the current run's service period rather than the originating period.** — Downstream invoicing only handles 'current invoice period only' line shapes. Period-preserving corrections require separate invoice-compatible correction handling that is not yet finished.
- **Credits are stripped from already-billed lines before subtraction, not before billing rating.** — Credit allocation happens after rating and varies independently of usage or pricing. Subtracting credit-inclusive lines would make credit changes appear as usage changes and produce incorrect deltas.

## Example: Add a new price type to the delta engine test harness

```
runDeltaRatingTestCase(t, deltaRatingTestCase{
    price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(10)}),
    phases: []deltaRatingPhase{
        {
            period:          periods.period1,
            meteredQuantity: 5,
            expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
                {
                    ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
                    Category:              stddetailedline.CategoryRegular,
                    ServicePeriod:         lo.ToPtr(periods.period1),
                    PerUnitAmount:         10, Quantity: 5,
                    Totals: ratingtestutils.ExpectedTotals{Amount: 50, Total: 50},
                },
            },
// ...
```

<!-- archie:ai-end -->
