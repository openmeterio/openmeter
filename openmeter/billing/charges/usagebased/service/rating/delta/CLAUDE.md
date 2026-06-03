# delta

<!-- archie:ai-start -->

> Production delta rating engine for usage-based charges: rates the current cumulative meter snapshot via billingrating.Service, subtracts all already-billed DetailedLines, and books the remaining delta on the current run's service period. Corrections are NOT attributed back to their originating period.

## Patterns

**Cumulative-snapshot subtraction** — Engine.Rate always calls ratingService.GenerateDetailedLines on the full cumulative quantity, then subtract.SubtractRatedRunDetails removes what was already billed. Never compute deltas at the meter-query level. (`billingDetailedLines, _ := e.ratingService.GenerateDetailedLines(usagebased.RateableIntent{...}, opts...); remaining, _ := subtract.SubtractRatedRunDetails(current, alreadyBilled, uniqueReferenceIDGenerator{})`)
**Credit stripping before subtraction** — Strip CreditsApplied and CreditsTotal from already-billed lines (after Clone) before SubtractRatedRunDetails so credit-allocation changes do not appear as usage/pricing changes. Also pass billingrating.WithCreditsMutatorDisabled() to GenerateDetailedLines. (`line = line.Clone(); line.CreditsApplied = nil; line.Totals.CreditsTotal = alpacadecimal.Zero; line.Totals.Total = line.Totals.CalculateTotal()`)
**Minimum commitment deferred to final period** — Append billingrating.WithMinimumCommitmentIgnored() whenever CurrentPeriod.ServicePeriod.To is before Intent.ServicePeriod.To. Minimum commitment applies only on the final snapshot. (`if in.CurrentPeriod.ServicePeriod.To.Before(in.Intent.ServicePeriod.To) { opts = append(opts, billingrating.WithMinimumCommitmentIgnored()) }`)
**All delta lines stamped to current period** — After subtraction, set every remaining line's ServicePeriod to CurrentPeriod.ServicePeriod and clear CorrectsRunID. Delta rating does not preserve the originating period. (`remaining[idx].ServicePeriod = in.CurrentPeriod.ServicePeriod; remaining[idx].CorrectsRunID = nil`)
**Sort then dense index then uniqueness check** — Call remaining.Sort() before assigning Index (persistence keys), then validate all ChildUniqueReferenceID values are unique (lo.GroupBy), erroring on duplicates. (`remaining.Sort(); for idx := range remaining { i := idx; remaining[idx].Index = &i }`)
**PreviousOnlyReversal requires persisted line ID** — uniqueReferenceIDGenerator.PreviousOnlyReversal errors if line.ID is empty; correction child ref is deterministic and stable. Always ensure already-billed lines carry their persisted ID. (`return fmt.Sprintf("%s#correction:detailed_line_id=%s", line.PricerReferenceID, line.ID), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Single public entry point Engine.Rate (New(ratingService)). Owns the full pipeline: validate, rate cumulative, convert to usagebased.DetailedLines, strip credits, subtract, stamp current period, sort, index, validate uniqueness. | Credits must be stripped before SubtractRatedRunDetails; WithCreditsMutatorDisabled() must also be passed to GenerateDetailedLines. |
| `uniquereferenceid.go` | Implements subtract.UniqueReferenceIDGenerator. CurrentOnly/MatchedDelta pass ChildUniqueReferenceID through unchanged; PreviousOnlyReversal builds a deterministic correction reference from the persisted line ID. | PreviousOnlyReversal errors on empty line.ID. The format '<PricerReferenceID>#correction:detailed_line_id=<ID>' must not change without updating downstream parsers. |
| `base_test.go` | deltaRatingTestCase / deltaRatingPhase table-driven harness; runDeltaRatingTestCase feeds prior-phase booked lines as AlreadyBilledDetailedLines to later phases. | detailedLinesBookedForDeltaTest assigns synthetic IDs ('phase-N-line-M') so reversals get stable correction refs. |
| `engine_test.go` | Unit tests using stubRatingService to control billing-rating output; covers credit stripping, partial-run minimum-commitment suppression, subtraction, correction refs. | stubRatingService.lastOpts asserts IgnoreMinimumCommitment and DisableCreditsMutator were set correctly. |

## Anti-Patterns

- Passing credit-inclusive AlreadyBilledDetailedLines to SubtractRatedRunDetails without zeroing CreditsApplied/CreditsTotal — credit changes appear as spurious usage deltas.
- Assigning Index values before calling Sort — indexes are persistence keys and must reflect final sort order.
- Setting CorrectsRunID on output lines — delta rating intentionally books all corrections on the current run period.
- Applying minimum commitment on partial runs — call WithMinimumCommitmentIgnored() whenever CurrentPeriod.To < Intent.ServicePeriod.To.
- Constructing Engine with a nil billingrating.Service — rating is the source of the cumulative line list subtraction operates on.

## Decisions

- **Delta engine books all corrections on the current run's service period rather than the originating period.** — Downstream invoicing only handles 'current invoice period only' line shapes; period-preserving corrections need separate invoice-compatible handling not yet finished.
- **Credits are stripped from already-billed lines before subtraction, not before billing rating.** — Credit allocation happens after rating and varies independently of usage/pricing; subtracting credit-inclusive lines would make credit changes look like usage changes.

## Example: Run a multi-phase delta rating test case

```
runDeltaRatingTestCase(t, deltaRatingTestCase{
  price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(10)}),
  phases: []deltaRatingPhase{{
    period: periods.period1, meteredQuantity: 5,
    expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{{
      ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
      Category: stddetailedline.CategoryRegular, ServicePeriod: lo.ToPtr(periods.period1),
      PerUnitAmount: 10, Quantity: 5,
      Totals: ratingtestutils.ExpectedTotals{Amount: 50, Total: 50},
    }},
  }},
})
```

<!-- archie:ai-end -->
