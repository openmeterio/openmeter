# rating

<!-- archie:ai-start -->

> Stateless usage-rating orchestrator for usage-based charges: snapshots metered quantity from ClickHouse (via streaming.Connector) at a stored-at cutoff and dispatches to a sub-engine (delta production engine / periodpreserving experimental engine, both built on the subtract primitive) to produce DetailedLines or totals. No DB writes — persistence is owned by callers in the run package.

## Patterns

**Config-struct constructor with Validate()** — New(Config) validates required fields before constructing the service, and every exported input type implements Validate() called at the top of the method body. (`func New(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**StoredAtLT cutoff in every ClickHouse query** — snapshotQuantity always sets FilterStoredAt.Lt = &in.StoredAtLT in the QueryMeter call so usage is bounded to a deterministic point in time, enabling idempotent re-rating. (`FilterStoredAt: &filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lt: &in.StoredAtLT}}`)
**Engine dispatch via charge.State.RatingEngine** — GetDetailedRatingForUsage switches on charge.State.RatingEngine (RatingEngineDelta or RatingEnginePeriodPreserving) with a default error case — no silent fallback. (`switch charge.State.RatingEngine { case usagebased.RatingEngineDelta: ... default: return ..., fmt.Errorf("unsupported rating engine: %s", ...) }`)
**Voided + current-run exclusion before subtraction** — eligibleRealizations filters out IsVoidedBillingHistory() runs and keeps only runs with ServicePeriodTo strictly before the current ServicePeriodTo, so the current run is never subtracted from itself. (`lo.Filter(charge.Realizations, func(run usagebased.RealizationRun, _ int) bool { if run.IsVoidedBillingHistory() { return false }; return run.ServicePeriodTo.Before(in.ServicePeriodTo) })`)
**Lazy DetailedLines loading via fetcher interface** — ensureDetailedLinesLoadedForRating only calls detailedLinesFetcher.FetchDetailedLines when some prior eligible run lacks DetailedLines.IsPresent(); rating refuses to proceed with incomplete prior runs. (`if !lo.EveryBy(charge.Realizations, func(run ...) bool { ... return run.DetailedLines.IsPresent() }) { s.detailedLinesFetcher.FetchDetailedLines(ctx, charge) }`)
**WithCreditsMutatorDisabled() on all rating calls** — Both GetTotalsForUsage and GetDetailedRatingForUsage pass billingrating.WithCreditsMutatorDisabled() so credit allocation is deferred to run creation, not applied during raw rating. (`opts := []billingrating.GenerateDetailedLinesOption{billingrating.WithCreditsMutatorDisabled()}`)
**Totals-only fast path** — GetTotalsForUsage skips detailed-line construction and returns ratingResult.Totals — preferred for pre-advance checks where only totals are needed. (`totals, err := svc.GetTotalsForUsage(ctx, GetTotalsForUsageInput{Charge: charge, StoredAtLT: storedAt})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (GetTotalsForUsage, GetDetailedRatingForUsage, GetPreferredRatingEngineFor), Config + Validate, and New constructor building deltaRater and periodPreservingRater. | Pure computation — no Ent/DB dependency. DetailedLinesFetcher is an injected interface; never inject an adapter directly. |
| `details.go` | GetDetailedRatingForUsage: ensureDetailedLinesLoadedForRating, snapshotQuantity, then dispatch to deltaRater or ratePeriodPreservingDetails. | The current run (ServicePeriodTo == run.ServicePeriodTo) is excluded from eligibleRealizations — never include it as a prior run. |
| `totals.go` | GetTotalsForUsage: snapshots quantity, calls GenerateDetailedLines with credits-mutator-disabled + optional ignore-minimum-commitment, returns ratingResult.Totals only. | Does NOT suffix ChildUniqueReferenceIDs — that belongs in the delta/periodpreserving engines. |
| `quantitysnapshot.go` | Private snapshotQuantity helper building QueryMeter params with FilterStoredAt + group-by filters and summing rows. | Validation errors are wrapped as billing.ValidationError{Err: err}; reuse that wrapper for new validations. |

## Anti-Patterns

- Adding Ent/DB adapter calls inside this package — persistence is exclusively the caller's responsibility.
- Calling snapshotQuantity without a non-zero StoredAtLT — every ClickHouse query must be stored-at bounded for idempotent re-rating.
- Including the current run in eligibleRealizations — it will be subtracted from itself and produce a zero bill.
- Passing voided realizations as AlreadyBilledDetailedLines — IsVoidedBillingHistory() runs must be filtered out before subtraction.
- Omitting WithCreditsMutatorDisabled() when calling GenerateDetailedLines — credit allocation must be deferred to run creation.

## Decisions

- **Stateless package with no DB dependency.** — Rating is pure computation (snapshot usage + apply rate card), making it trivially testable with MockStreamingConnector and reusable across callers without transaction concerns.
- **Engine selection at dispatch time via charge.State.RatingEngine.** — Delta is production-safe while period-preserving is experimental; the charge carries its own engine preference without the caller knowing engine internals.
- **Lazy DetailedLines loading via a fetcher interface rather than caller pre-loading.** — Prior runs are usually already expanded; the fetcher avoids redundant DB round-trips while providing a safe fallback when lines are missing.

## Example: Rate a usage-based charge and retrieve detailed lines with the stored-at cutoff

```
svc, err := usagebasedrating.New(usagebasedrating.Config{
    StreamingConnector:   streamingConnector,
    RatingService:        billingratingservice.New(),
    DetailedLinesFetcher: detailedLinesFetcher,
})
if err != nil { return err }
result, err := svc.GetDetailedRatingForUsage(ctx, usagebasedrating.GetDetailedRatingForUsageInput{
    Charge:          charge,          // must have State.RatingEngine set
    ServicePeriodTo: currentPeriodTo, // within Charge.Intent.ServicePeriod
})
```

<!-- archie:ai-end -->
