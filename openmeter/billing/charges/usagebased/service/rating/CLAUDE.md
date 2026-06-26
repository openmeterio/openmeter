# rating

<!-- archie:ai-start -->

> Rating orchestration layer for usage-based charges: snapshots metered quantity at a stored-at cutoff, then dispatches to one of two delta-rating engines (delta or periodpreserving) to produce gross detailed lines and totals for a charge run. It is the bridge between the streaming meter, billing rating, and the per-period delta engines in its subpackages.

## Patterns

**Service interface + private struct + Config-validated New** — Public Service interface (GetTotalsForUsage, GetDetailedRatingForUsage, GetPreferredRatingEngineFor) implemented by unexported *service; New(Config) validates non-nil deps via Config.Validate() before constructing. (`func New(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err }; ... }`)
**Input struct with Validate() before any work** — Each entry point takes a typed Input struct whose Validate() runs first; service period bounds must satisfy From < ServicePeriodTo <= Intent.ServicePeriod.To and StoredAtLT must be non-zero. (`func (i GetDetailedRatingForUsageInput) Validate() error { if !i.ServicePeriodTo.After(period.From) {...}; if i.StoredAtLT.IsZero() {...} }`)
**Engine dispatch on charge.State.RatingEngine** — GetDetailedRatingForUsage switches on charge.State.RatingEngine: RatingEngineDelta -> deltaRater.Rate, RatingEnginePeriodPreserving -> ratePeriodPreservingDetails; unknown engine returns an error. Both engines are constructed in New (delta.New/periodpreserving.New). (`switch charge.State.RatingEngine { case usagebased.RatingEngineDelta: ... case usagebased.RatingEnginePeriodPreserving: ... default: return ..., fmt.Errorf("unsupported rating engine: %s", ...) }`)
**Credits mutator always disabled for gross rating** — Rating here must stay gross; the totals path passes billingrating.WithCreditsMutatorDisabled() because run creation applies credits later. The delta/periodpreserving engines also strip credits internally. (`opts := []billingrating.GenerateDetailedLinesOption{ billingrating.WithCreditsMutatorDisabled() }`)
**Voided realizations excluded from billable history** — Eligible prior runs are filtered with run.IsVoidedBillingHistory() == false and ServicePeriodTo strictly before current ServicePeriodTo; voided runs are skipped for both detailed-line loading and delta subtraction. (`eligibleRealizations := lo.Filter(charge.Realizations, func(run usagebased.RealizationRun, _ int) bool { if run.IsVoidedBillingHistory() { return false }; return run.ServicePeriodTo.Before(in.ServicePeriodTo) })`)
**Lazy prior detailed-line expansion with overcharge guard** — ensureDetailedLinesLoadedForRating only calls detailedLinesFetcher when a non-voided prior run before the cutoff lacks detailed lines, then re-asserts all eligible priors are expanded — rating refuses to proceed on incomplete prior runs to avoid overcharging. (`if run.ServicePeriodTo.Before(servicePeriodTo) && !run.DetailedLines.IsPresent() { return ..., fmt.Errorf("prior runs[%d]: detailed lines must be expanded", idx) }`)
**Quantity snapshot via streaming with stored-at cutoff** — snapshotQuantity builds streaming.QueryParams with FilterCustomer, From/To from the service period, FilterGroupBy from MeterGroupByFilters, and FilterStoredAt Lt = StoredAtLT, then sums rows via summarizeMeterQueryRow. (`FilterStoredAt: &filter.FilterTimeUnix{ FilterTime: filter.FilterTime{ Lt: &in.StoredAtLT } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, Config + Validate, *service struct, New constructor wiring deltaRater/periodPreservingRater and the DetailedLinesFetcher interface. | GetPreferredRatingEngineFor currently always returns RatingEngineDelta regardless of intent; do not assume periodpreserving is auto-selected. |
| `details.go` | GetDetailedRatingForUsage: loads prior lines, snapshots current quantity, dispatches to delta or periodpreserving, builds PriorPeriod cumulative snapshots. | currentBillingPeriod advances From past the latest eligible realization's ServicePeriodTo; prior-period meter queries are cumulative from Intent.ServicePeriod.From, while PriorPeriod.ServicePeriod is only the billing slice. |
| `quantitysnapshot.go` | getQuantityForUsage / snapshotQuantity / summarizeMeterQueryRow — the only place meters are queried; wraps validation failures in billing.ValidationError. | summarizeMeterQueryRow sums all rows as float via alpacadecimal.NewFromFloat(row.Value); precision loss originates here, not in the engines. |
| `totals.go` | GetTotalsForUsage: fast path that snapshots quantity then calls ratingService.GenerateDetailedLines directly, returning only Totals. | Always passes WithCreditsMutatorDisabled; WithMinimumCommitmentIgnored is conditional on IgnoreMinimumCommitment. Bypasses the delta/periodpreserving engines entirely. |
| `service_test.go` | Behavior tests using stubRatingService, MockStreamingConnector, billingratingservice.New(), and ratingtestutils expectation helpers (ToExpectedDetailedLinesWithServicePeriod, ToExpectedTotals). | Tests assert credits mutator disabled (lastOpts.DisableCreditsMutator), current-run-on-charge is ignored, and prior detailed lines are fetched exactly once. |

## Anti-Patterns

- Querying meters outside snapshotQuantity, or skipping the StoredAtLT/FilterStoredAt cutoff — stored-at consistency across current and prior snapshots is load-bearing.
- Treating voided realizations (IsVoidedBillingHistory) as previously invoiced periods — they must be excluded from eligible realizations and detailed-line loading.
- Proceeding to rate when a non-voided prior run before the cutoff has unexpanded detailed lines — this overcharges customers and the guard exists to prevent it.
- Rating without WithCreditsMutatorDisabled here — totals/lines must be gross because credits are applied later during run creation.
- Selecting the rating engine anywhere but via charge.State.RatingEngine, or assuming GetPreferredRatingEngineFor returns periodpreserving (it returns Delta).

## Decisions

- **Two interchangeable rating engines (delta, periodpreserving) selected per-charge via State.RatingEngine, both constructed in New.** — Delta rolls all corrections onto the current run period (invoice-safe today); periodpreserving keeps corrections on their original period but is not yet invoice-safe.
- **Prior-period quantities are re-queried cumulatively from Intent.ServicePeriod.From using the current run's StoredAtLT, not copied from stored prior snapshots.** — Rating engines need cumulative quantity for correct delta subtraction; a TODO documents a future optimization to reuse prior snapshots for monotonic meters.
- **GetTotalsForUsage bypasses the delta engines and calls billing rating directly.** — Totals do not require per-period detailed-line subtraction, so the fast path avoids generating detailed lines.

## Example: Snapshot metered quantity with a stored-at cutoff and dispatch to the delta engine.

```
currentQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
    Customer:      in.Customer.Customer,
    FeatureMeter:  in.FeatureMeter,
    ServicePeriod: currentRunServicePeriod,
    StoredAtLT:    in.StoredAtLT,
})
if err != nil { return GetDetailedRatingForUsageResult{}, fmt.Errorf("get current quantity: %w", err) }

out, err := s.deltaRater.Rate(ctx, delta.Input{
    Intent: charge.Intent,
    CurrentPeriod: delta.CurrentPeriod{
        MeteredQuantity: currentQuantity,
        ServicePeriod:   currentBillingPeriod(currentRunServicePeriod, eligibleRealizations),
    },
    AlreadyBilledDetailedLines: alreadyBilledDetailedLines,
// ...
```

<!-- archie:ai-end -->
