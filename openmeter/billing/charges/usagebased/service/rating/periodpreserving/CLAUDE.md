# periodpreserving

<!-- archie:ai-start -->

> Period-preserving rating engine for usage-based charges: rates cumulative meter snapshots for the current and all prior periods at the current stored-at cutoff, subtracts already-billed lines per period, and emits corrections that keep their source service period and CorrectsRunID. NOT production-ready for invoices until invoice-compatible correction handling exists (see README warning).

## Patterns

**Epoch-based cumulative rating** — Build rating epochs from sorted prior periods plus the current period. For each epoch, GenerateDetailedLines for the cumulative quantity up to that epoch's end, then subtract earlier-epoch lines to isolate lines attributable to that epoch. (`epochLines = subtract.SubtractRatedRunDetails(epochCumulative, priorEpochCumulative, generatedUniqueReferenceIDGenerator{}, subtract.WithUniqueReferenceIDValidationIgnored())`)
**Period-stamped ChildUniqueReferenceID** — Stamp each output line's ChildUniqueReferenceID with the source service period as '<id>@[<from>..<to>]' so child refs are unique within a run when a component appears across multiple periods. (`FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1)`)
**CorrectsRunID on prior-period corrections** — Subtractions for a prior period set CorrectsRunID on output corrections to the corrected run ID. Current-period lines leave CorrectsRunID nil. (`corrections[i].CorrectsRunID = lo.ToPtr(priorPeriod.RunID)`)
**Non-overlapping prior period validation** — Validate prior periods do not overlap each other or the current period, and are non-empty at streaming.MinimumWindowSizeDuration precision; overlaps produce duplicate period-stamped identities. (`if priorPeriod.ServicePeriod.Overlaps(i.CurrentPeriod.ServicePeriod) { errs = append(errs, ...) }`)
**Intermediate subtraction with validation ignored** — Epoch-to-epoch subtraction must pass WithUniqueReferenceIDValidationIgnored() because intermediate lines are not persisted and may carry duplicate refs before period stamping. Only the final stamped output gets uniqueness validation. (`subtract.SubtractRatedRunDetails(currentEpoch, previousEpoch, generatedUniqueReferenceIDGenerator{}, subtract.WithUniqueReferenceIDValidationIgnored())`)
**Credit stripping from already-billed lines** — Same as delta: strip CreditsApplied and CreditsTotal from already-billed lines before subtraction arithmetic. (`line.CreditsApplied = nil; line.Totals.CreditsTotal = alpacadecimal.Zero; line.Totals.Total = line.Totals.CalculateTotal()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Engine.Rate orchestrates the multi-epoch pipeline: validate, SortPriorPeriods, build epochs, generate cumulative lines per epoch, epoch-to-epoch subtract, strip credits and subtract already-billed per prior period, stamp period suffixes, sort, assign dense indexes. | Intermediate epoch subtraction ignores unique-ID validation; only the final stamped output is validated for uniqueness. Validation rejects overlapping or empty prior periods. |
| `uniquereferenceid.go` | Two generators: generatedUniqueReferenceIDGenerator (all methods pass through ChildUniqueReferenceID for intermediate epoch arithmetic) and bookedCorrectionUniqueReferenceIDGenerator (PreviousOnlyReversal builds '<PricerReferenceID>#correction:detailed_line_id=<ID>'). | bookedCorrectionUniqueReferenceIDGenerator.PreviousOnlyReversal requires line.ID; generatedUniqueReferenceIDGenerator is safe with empty IDs. |
| `engine_test.go` | lateEventRatingTestCase / lateEventRatingPhase harness with usagePerPhaseCumulative simulating snapshots arriving across multiple runs; covers late-event and repricing scenarios. | README warns this engine is not invoice-ready; invoice-level correction handling (negative commitment/discount lines) is unfinished. |

## Anti-Patterns

- Using this engine as the invoice-facing rater — min/max commitments and discounts can emit negative commitment/discount lines downstream invoicing cannot handle.
- Validating ChildUniqueReferenceID uniqueness on intermediate epoch-subtraction outputs — use WithUniqueReferenceIDValidationIgnored() and only validate final stamped output.
- Allowing overlapping prior periods — period stamping produces the same identity for overlapping buckets, causing duplicate ChildUniqueReferenceID errors.
- Setting CorrectsRunID on current-period lines — only prior-period correction output gets CorrectsRunID.
- Skipping credit stripping before already-billed subtraction — credit allocation changes produce spurious usage/pricing deltas.

## Decisions

- **Corrections keep their originating service period and CorrectsRunID instead of being rolled into the current run period.** — Late-arriving usage must be attributed to the period where it occurred; invoice clients need period-accurate correction lines.
- **Engine is explicitly marked not production-ready for invoice use in README.** — Post-rating mutations (min/max commitment, discounts) can produce mathematically correct but invoice-incompatible negative lines.

<!-- archie:ai-end -->
