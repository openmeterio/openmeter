# periodpreserving

<!-- archie:ai-start -->

> Period-preserving rating engine for usage-based charges: computes detailed-line deltas while keeping corrections on the service period where the corrected usage belongs (late-arriving usage / repricing against prior periods). NOT yet invoice-safe — must not be used as the production invoice-facing engine until invoice-compatible correction handling lands.

## Patterns

**Engine{ratingService} via New; Rate(ctx, Input) entry point** — Same shape as delta: value Engine holding billingrating.Service, built with New. Input carries Intent, CurrentPeriod, and a slice of PriorPeriod (each with RunID, MeteredQuantity, ServicePeriod, DetailedLines). (`engine := New(billingratingservice.New()); out, err := engine.Rate(ctx, Input{Intent: intent, CurrentPeriod: ..., PriorPeriods: ...})`)
**Strict period validation: containment, non-empty-at-window-precision, no overlap** — Input.Validate joins errors: every period must be ContainsPeriodInclusive of the intent period; prior periods must not be empty when Truncate(streaming.MinimumWindowSizeDuration); prior periods must not Overlaps the current period or each other. (`if priorPeriod.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() { errs = append(errs, ...) }`)
**Epoch-based cumulative rating with second-precision epoch keys** — buildDetailsByEpoch sorts prior+current into epochs keyed by epochClosedPeriod{From,To int64} (Unix seconds; sub-second dropped intentionally). For each epoch it rates the cumulative quantity and subtracts previously-generated lines to isolate new lines, preserving each line's source period. (`result[servicePeriod] = append(result[servicePeriod], line) // grouped by epochClosedPeriod`)
**Allow duplicate child refs in intermediate subtraction; period-stamp at the end** — Epoch subtraction passes subtract.WithUniqueReferenceIDValidationIgnored() because intermediate output is not persisted. flattenDetailedLinesByEpoch calls WithServicePeriodFromUniqueReferenceID() to stamp the period suffix, Sort()s, and assigns dense Index. The period suffix makes the same rating component distinct across periods. (`subtract.SubtractRatedRunDetails(cur, prev, generatedUniqueReferenceIDGenerator{}, subtract.WithUniqueReferenceIDValidationIgnored())`)
**Set CorrectsRunID on prior-period outputs** — After subtracting already-billed lines per prior period, every output line for that period gets CorrectsRunID = lo.ToPtr(runID.ID). This is the key difference from delta, which clears CorrectsRunID. (`result[servicePeriod][idx].CorrectsRunID = lo.ToPtr(runID.ID)`)
**Two distinct reference-ID generators** — generatedUniqueReferenceIDGenerator (intermediate, pass-through child refs) for epoch subtraction; bookedCorrectionUniqueReferenceIDGenerator (PreviousOnlyReversal encodes "<pricerRef>#correction:detailed_line_id=<id>", erroring on missing ID) for subtracting already-billed lines. (`subtract.SubtractRatedRunDetails(result[servicePeriod], alreadyBilled, bookedCorrectionUniqueReferenceIDGenerator{})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Whole engine: Input/CurrentPeriod/PriorPeriod/Result, Validate, SortPriorPeriods, Rate, buildDetailsByEpoch, flattenDetailedLinesByEpoch, epochClosedPeriod helpers. | epochClosedPeriod uses Unix-second keys — sub-second period boundaries collapse; period validation rejects overlapping prior periods because overlaps leak duplicate period-stamped child refs to persistence. Credits are stripped from already-billed lines before subtraction (same as delta). |
| `uniquereferenceid.go` | generatedUniqueReferenceIDGenerator and bookedCorrectionUniqueReferenceIDGenerator implementing subtract.UniqueReferenceIDGenerator. | bookedCorrectionUniqueReferenceIDGenerator.PreviousOnlyReversal requires line.ID; it errors 'detailed line id is required' otherwise. The generated generator is pure pass-through. |
| `engine_test.go` | Multi-phase late-event scenarios (runLateEventRatingTestCase) using a real billingratingservice.New(); covers unit price, max/min commitments, percentage and usage discounts kept on original periods. | Expected lines assert FormatDetailedLineChildUniqueReferenceID(...,period) suffixes and CorrectsRunID pointers; the README warns these mutation cases (negative discount/commitment deltas) are not yet invoice-safe. |
| `README.md` | Algorithm, period-stamping rules, examples, and an explicit Warning + TODO list of unfinished invoice-compatible correction cases. | Treat the Warning as binding: do not wire this engine as the production invoice rater yet. |

## Anti-Patterns

- Using this engine as the production invoice-facing rater before invoice-compatible correction handling exists (README Warning).
- Validating uniqueness on intermediate epoch subtraction — duplicate child refs are expected there and only resolved by period-stamping.
- Allowing overlapping prior periods or empty-at-window-precision periods — both are rejected and would corrupt period-stamped identities.
- Forgetting to set CorrectsRunID on prior-period corrections (this is what distinguishes period-preserving from delta).
- Keying epochs at sub-second precision — epochClosedPeriod is deliberately Unix-second granularity.

## Decisions

- **Keep corrections on the original service period and mark the corrected run via CorrectsRunID instead of rolling into the current period.** — Invoices must show changes against the prior period for late-arriving usage and repricing; delta rating cannot express that.
- **Encode the service period into ChildUniqueReferenceID as a suffix and only validate uniqueness on final, period-stamped output.** — The same rating component appears across multiple periods; the period suffix gives each persisted identity a distinct, collision-free key within the run.
- **Use Unix-second epoch keys.** — Meter snapshots are evaluated at streaming minimum-window precision; sub-second boundaries carry no rating meaning and would fragment epoch grouping.

## Example: Per-epoch cumulative rating, isolate new lines, then subtract already-billed and stamp CorrectsRunID

```
periodNew, err := subtract.SubtractRatedRunDetails(detailedLinesWithUsageFromPriorPeriods, previouslyGeneratedDetailedLines, generatedUniqueReferenceIDGenerator{}, subtract.WithUniqueReferenceIDValidationIgnored())
// ...later, per prior period:
remaining, err := subtract.SubtractRatedRunDetails(result[servicePeriod], alreadyBilled, bookedCorrectionUniqueReferenceIDGenerator{})
for idx := range result[servicePeriod] { result[servicePeriod][idx].CorrectsRunID = lo.ToPtr(runID.ID) }
```

<!-- archie:ai-end -->
