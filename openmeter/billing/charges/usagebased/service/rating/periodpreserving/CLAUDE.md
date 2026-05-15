# periodpreserving

<!-- archie:ai-start -->

> Implements the period-preserving rating engine for usage-based charges: rates cumulative meter snapshots for the current period and all prior periods with the current stored-at cutoff, subtracts already-billed detailed lines per period, and emits corrections that keep their source service period and CorrectsRunID. NOT safe for production invoice use until invoice-compatible correction handling is finished (see README warning).

## Patterns

**Epoch-based cumulative rating** — Build rating epochs from sorted prior periods plus the current period. For each epoch, call billingrating.Service.GenerateDetailedLines for the cumulative quantity up to that epoch's end. Subtract all earlier-epoch generated lines to isolate new lines attributable to that epoch. (`For each epoch: billingDetailedLines = ratingService.GenerateDetailedLines(RateableIntent{...MeterValue: cumulativeUpToEpoch}); epochLines = SubtractRatedRunDetails(epochCumulative, priorEpochCumulative, generatedUniqueReferenceIDGenerator{}, WithUniqueReferenceIDValidationIgnored())`)
**Period-stamped ChildUniqueReferenceID** — Before returning output lines, stamp each line's ChildUniqueReferenceID with the source service period using the format '<id>@[<from>..<to>]'. This makes child references unique within a run when the same rating component appears across multiple periods. (`FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1) == 'unit-price-usage@[2025-01-01T00:00:00Z..2025-01-10T00:00:00Z]'`)
**CorrectsRunID on prior-period corrections** — When subtracting already-billed lines for a prior period, set CorrectsRunID on output correction lines to the corrected realization run ID. Current-period lines leave CorrectsRunID nil. (`for _, priorPeriod := range sortedPriorPeriods { corrections := subtractAlreadyBilledForPeriod(...); for i := range corrections { corrections[i].CorrectsRunID = lo.ToPtr(priorPeriod.RunID) } }`)
**Non-overlapping prior period validation** — Validate that prior periods do not overlap each other or the current period before rating. Overlapping buckets produce the same period-stamped identity twice, causing duplicate ChildUniqueReferenceID errors. (`if err := i.CurrentPeriod.ServicePeriod.Validate(); err != nil { ... }; // also check all priorPeriods against each other`)
**Intermediate subtraction with validation ignored** — Epoch-to-epoch subtraction for building expected lines per epoch must pass WithUniqueReferenceIDValidationIgnored() because intermediate lines are not persisted and may have duplicate references before period stamping. (`subtract.SubtractRatedRunDetails(currentEpoch, previousEpoch, generatedUniqueReferenceIDGenerator{}, subtract.WithUniqueReferenceIDValidationIgnored())`)
**Credit stripping from already-billed lines** — Same as delta engine — strip CreditsApplied and CreditsTotal from already-billed lines before subtraction arithmetic so credit-allocation changes do not appear as usage changes. (`line.CreditsApplied = nil; line.Totals.CreditsTotal = alpacadecimal.Zero; line.Totals.Total = line.Totals.CalculateTotal()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Engine.Rate orchestrates the multi-epoch pipeline: validates input, sorts prior periods, builds epochs, generates cumulative lines per epoch, does epoch-to-epoch subtraction, strips credits and subtracts already-billed per prior period, stamps output with period suffixes, sorts, assigns indexes. | The intermediate epoch subtraction ignores unique-ID validation (WithUniqueReferenceIDValidationIgnored) — only the final stamped output gets uniqueness validation. Do not accidentally validate intermediate lines. |
| `uniquereferenceid.go` | Provides two generators: generatedUniqueReferenceIDGenerator (all three methods pass through ChildUniqueReferenceID for intermediate epoch arithmetic) and bookedCorrectionUniqueReferenceIDGenerator (PreviousOnlyReversal generates '<PricerReferenceID>#correction:detailed_line_id=<ID>' for already-billed corrections). | bookedCorrectionUniqueReferenceIDGenerator.PreviousOnlyReversal requires line.ID — it errors if the already-billed line has no persisted ID. generatedUniqueReferenceIDGenerator is safe to use with empty IDs. |
| `engine_test.go` | Comprehensive late-event and repricing scenarios using a lateEventRatingTestCase / lateEventRatingPhase harness. Each phase provides usagePerPhaseCumulative (per-snapshot cumulative quantities per period) so the test engine can simulate meter snapshots arriving across multiple runs. | The README warns this engine is NOT production-ready for invoices. All tests exercise internal correctness but invoice-level correction handling (negative commitment/discount lines) is unfinished. |

## Anti-Patterns

- Do not use this engine as the invoice-facing rater — minimum commitments, maximum commitments, usage discounts, and percentage discounts can emit negative commitment/discount lines that downstream invoice systems cannot handle.
- Do not validate ChildUniqueReferenceID uniqueness on intermediate epoch-subtraction outputs — call WithUniqueReferenceIDValidationIgnored() for intermediate steps and only validate the final period-stamped output.
- Do not allow overlapping prior periods — period stamping produces the same identity for overlapping buckets, causing duplicate ChildUniqueReferenceID errors.
- Do not set CorrectsRunID on current-period lines — only prior-period correction output gets CorrectsRunID set to the corrected run ID.
- Do not skip credit stripping before already-billed subtraction — credit allocation changes will produce spurious usage or pricing deltas.

## Decisions

- **Corrections keep their originating service period and CorrectsRunID instead of being rolled into the current run period.** — Late-arriving usage must be attributed to the period where the usage occurred. Invoice clients need period-accurate correction lines to show customers where their usage was revised.
- **Engine is explicitly marked not production-ready for invoice use in README.** — Post-rating mutations (min/max commitment, discounts) can produce mathematically correct but invoice-incompatible negative lines. Invoice-compatible correction handling must be added before this engine faces customers.

<!-- archie:ai-end -->
