# delta

<!-- archie:ai-start -->

> Delta rating engine for usage-based charges: it rates the latest cumulative meter snapshot, subtracts every detailed line already booked for the charge, and books the remaining delta on the current run's service period. It intentionally does NOT preserve the original service period of corrections — every delta lands on the current run period.

## Patterns

**Engine{ratingService} constructed via New** — The engine is a value type holding a billingrating.Service, constructed with New(ratingService). Rate(ctx, Input) is the only exported method. (`engine := New(billingratingservice.New()); out, err := engine.Rate(ctx, Input{Intent: intent, CurrentPeriod: CurrentPeriod{...}, AlreadyBilledDetailedLines: ...})`)
**Validate before rating** — Rate calls in.Validate() first; Input.Validate collects errors into []error and returns models.NewNillableGenericValidationError(errors.Join(errs...)). Current period service period must be ContainsPeriodInclusive of the intent service period. (`if err := in.Validate(); err != nil { return Result{}, err }`)
**Disable credits mutator; ignore min commitment until final snapshot** — Always pass billingrating.WithCreditsMutatorDisabled(); add billingrating.WithMinimumCommitmentIgnored() only when CurrentPeriod.ServicePeriod.To.Before(Intent.ServicePeriod.To). Minimum commitment is charged solely on the final service-period snapshot. (`opts := []billingrating.GenerateDetailedLinesOption{billingrating.WithCreditsMutatorDisabled()}`)
**Strip credits from already-billed lines before subtraction** — Clone each AlreadyBilledDetailedLine, set CreditsApplied=nil, Totals.CreditsTotal=Zero, recompute Totals.Total via CalculateTotal(). Credits are allocated post-rating, so they must not look like usage/pricing changes during subtraction. (`line = line.Clone(); line.CreditsApplied = nil; line.Totals.CreditsTotal = alpacadecimal.Zero; line.Totals.Total = line.Totals.CalculateTotal()`)
**Stamp current period and clear CorrectsRunID on output** — After subtract.SubtractRatedRunDetails, every remaining line gets ServicePeriod = CurrentPeriod.ServicePeriod and CorrectsRunID = nil. The period-preserving engine owns correction metadata; delta never sets it. (`remainingDetailedLines[idx].ServicePeriod = in.CurrentPeriod.ServicePeriod; remainingDetailedLines[idx].CorrectsRunID = nil`)
**Sort, assign dense indexes, validate unique child references** — Call remainingDetailedLines.Sort(), then assign Index = &idx densely. Finally group by ChildUniqueReferenceID and error on any duplicate — delta output (unlike periodpreserving intermediate steps) must be unique. (`remainingDetailedLines.Sort(); for idx := range ... { index := idx; remainingDetailedLines[idx].Index = &index }`)
**uniqueReferenceIDGenerator{} for subtraction identities** — Rate passes a local uniqueReferenceIDGenerator{} (in uniquereferenceid.go) implementing subtract.UniqueReferenceIDGenerator to encode current/matched/reversal child reference identities. (`subtract.SubtractRatedRunDetails(currentDetailedLines, alreadyBilledDetailedLines, uniqueReferenceIDGenerator{})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Engine, Input/CurrentPeriod/Result types, and Rate — the whole delta algorithm. Input.Validate enforces current period containment. | Do not preserve correction periods here; ServicePeriod is overwritten to the current run period and CorrectsRunID is cleared. Duplicate output child refs are a hard error. |
| `uniquereferenceid.go` | uniqueReferenceIDGenerator implementing subtract.UniqueReferenceIDGenerator (CurrentOnly/MatchedDelta/PreviousOnlyReversal). | PreviousOnlyReversal must produce a deterministic correction ID like "<pricerRef>#correction:detailed_line_id=<id>"; it errors when the already-billed detailed line ID is missing. |
| `engine_test.go` | Stub-driven unit tests using stubRatingService (records lastOpts, returns canned DetailedLines). | Tests assert lastOpts.DisableCreditsMutator and IgnoreMinimumCommitment flags — keep the option-passing logic intact when editing Rate. |
| `base_test.go` | runDeltaRatingTestCase harness driving multi-phase scenarios via a real billingratingservice.New(); deltaRatingTestPeriods() supplies period1..3. | Each phase feeds prior phases' booked lines as AlreadyBilledDetailedLines; detailedLinesBookedForDeltaTest stamps phase-N-line-M IDs used by correction refs. |
| `dynamic_test.go / unit_test.go / package_test.go / tieredgraduated_test.go / tieredvolume_test.go` | Price-shape-specific delta scenarios (dynamic, unit, package, graduated, volume) including repricing corrections. | Expected lines assert exact ChildUniqueReferenceID correction strings (e.g. "usage#correction:detailed_line_id=phase-1-line-1"); changing the generator format breaks many tests. |

## Anti-Patterns

- Preserving the original service period of a correction — delta always restamps to the current run period.
- Setting CorrectsRunID on delta output (it is always cleared to nil).
- Calling billing rating without WithCreditsMutatorDisabled(), or leaving credits on already-billed lines before subtraction.
- Returning on the first validation error instead of joining via models.NewNillableGenericValidationError.
- Emitting duplicate ChildUniqueReferenceID values — delta validates uniqueness and errors out.

## Decisions

- **Roll every delta onto the current run period instead of preserving correction periods.** — Keeps downstream invoicing in the simpler 'current invoice period only' shape while period-preserving rating and invoice correction support mature (see README and periodpreserving sibling).
- **Compare per-unit amount by decimal equality (delegated to subtract) so repricing emits a reversal + current line.** — Non-linear prices (volume tiers) can re-rate the whole quantity; treating it as a quantity delta would be wrong.

## Example: Rate the cumulative snapshot, strip credits, subtract already-billed, restamp to current period

```
billingDetailedLines, err := e.ratingService.GenerateDetailedLines(usagebased.RateableIntent{Intent: in.Intent, ServicePeriod: in.CurrentPeriod.ServicePeriod, MeterValue: in.CurrentPeriod.MeteredQuantity}, billingrating.WithCreditsMutatorDisabled())
current := usagebased.NewDetailedLinesFromBilling(in.Intent, in.CurrentPeriod.ServicePeriod, billingDetailedLines.DetailedLines)
remaining, err := subtract.SubtractRatedRunDetails(current, alreadyBilled, uniqueReferenceIDGenerator{})
for idx := range remaining { remaining[idx].ServicePeriod = in.CurrentPeriod.ServicePeriod; remaining[idx].CorrectsRunID = nil }
```

<!-- archie:ai-end -->
