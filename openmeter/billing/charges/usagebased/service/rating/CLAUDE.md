# rating

<!-- archie:ai-start -->

> Stateless usage-rating orchestrator: snapshots metered quantity from ClickHouse (via streaming.Connector) and dispatches to the delta or period-preserving sub-engine to produce DetailedLines or totals. No DB writes — all persistence is owned by callers in the run package.

## Patterns

**Config-struct constructor with Validate()** — New(Config) validates all required fields before constructing the service. Every exported input type also implements Validate() and is called at the top of the method body. (`func New(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**StoredAtLT cutoff in every ClickHouse query** — snapshotQuantity always sets FilterStoredAt.Lt = &in.StoredAtLT in the QueryMeter call so usage is bounded to a deterministic point in time, enabling idempotent re-rating. (`FilterStoredAt: &filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lt: &in.StoredAtLT}}`)
**Rating-engine dispatch via charge.State.RatingEngine** — GetDetailedRatingForUsage switches on charge.State.RatingEngine (RatingEngineDelta or RatingEnginePeriodPreserving) to call deltaRater.Rate or periodPreservingRater.Rate. A default case returns an error — no silent fallback. (`switch charge.State.RatingEngine { case usagebased.RatingEngineDelta: ... case usagebased.RatingEnginePeriodPreserving: ... default: return ..., fmt.Errorf("unsupported rating engine: %s", ...) }`)
**Voided-realization exclusion before subtraction** — eligibleRealizations filters out runs where run.IsVoidedBillingHistory() before passing AlreadyBilledDetailedLines to the delta engine. Voided runs must also be skipped in ensureDetailedLinesLoadedForRating. (`lo.Filter(charge.Realizations, func(run usagebased.RealizationRun, _ int) bool { if run.IsVoidedBillingHistory() { return false } return run.ServicePeriodTo.Before(in.ServicePeriodTo) })`)
**Lazy detailed-line loading via DetailedLinesFetcher** — ensureDetailedLinesLoadedForRating checks whether all prior eligible runs have DetailedLines.IsPresent() before calling detailedLinesFetcher.FetchDetailedLines — avoiding unnecessary fetches when lines are already loaded. (`if !lo.EveryBy(charge.Realizations, func(run ...) bool { ... return run.DetailedLines.IsPresent() }) { expandedCharge, err := s.detailedLinesFetcher.FetchDetailedLines(ctx, charge) }`)
**WithCreditsMutatorDisabled() always set on rating calls** — Both GetTotalsForUsage and GetDetailedRatingForUsage pass billingrating.WithCreditsMutatorDisabled() to ratingService.GenerateDetailedLines so that credit allocation is not applied during rating (callers apply credits separately). (`opts := []billingrating.GenerateDetailedLinesOption{billingrating.WithCreditsMutatorDisabled()}`)
**Prefer GetTotalsForUsage over GetDetailedRatingForUsage when only totals are needed** — GetTotalsForUsage skips detailed-line construction and only calls ratingResult.Totals — materially faster for pre-advance checks. (`totals, err := svc.GetTotalsForUsage(ctx, GetTotalsForUsageInput{Charge: charge, StoredAtLT: storedAt, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the Service interface (GetTotalsForUsage, GetDetailedRatingForUsage, GetPreferredRatingEngineFor), Config struct with Validate(), and New constructor that builds deltaRater and periodPreservingRater. | Pure computation — no Ent/DB dependency. DetailedLinesFetcher is an interface injected via Config; never inject an adapter directly. |
| `details.go` | Implements GetDetailedRatingForUsage: calls ensureDetailedLinesLoadedForRating, snapshotQuantity, currentBillingPeriod, then dispatches to deltaRater or ratePeriodPreservingDetails. | The current run (ServicePeriodTo == run.ServicePeriodTo) is excluded from eligibleRealizations — never include it as a prior run or subtraction will zero out the current bill. |
| `totals.go` | Implements GetTotalsForUsage: snapshots quantity, calls ratingService.GenerateDetailedLines with DisableCreditsMutator+optional IgnoreMinimumCommitment, and returns only ratingResult.Totals. | Does NOT suffix ChildUniqueReferenceIDs — do not add that logic here; it belongs in the delta/period-preserving engines. |
| `quantitysnapshot.go` | Private snapshotQuantity helper: builds QueryMeter params with FilterStoredAt and MeterGroupByFilters, calls streaming.Connector, sums rows via summarizeMeterQueryRow. | Validation errors wrapped as billing.ValidationError{Err: err} — use the same wrapper for any new validations added here. |

## Anti-Patterns

- Adding Ent/DB adapter calls inside this package — persistence is exclusively the caller's responsibility.
- Calling snapshotQuantity without a non-zero StoredAtLT — every ClickHouse query must be bounded by the stored-at cutoff for idempotent re-rating.
- Including the current run (ServicePeriodTo == in.ServicePeriodTo) in eligibleRealizations — it will be subtracted from itself and produce a zero bill.
- Passing voided realizations to deltaRater.Rate as AlreadyBilledDetailedLines — IsVoidedBillingHistory() runs must be filtered out before subtraction.
- Omitting WithCreditsMutatorDisabled() when calling ratingService.GenerateDetailedLines — credit allocation must be deferred to run creation, not applied during raw rating.

## Decisions

- **Stateless package with no DB dependency** — Rating is a pure computation (snapshot usage + apply rate card). Keeping it DB-free makes it trivially testable with MockStreamingConnector and reusable from multiple callers without transaction concerns.
- **Rating-engine selection at dispatch time via charge.State.RatingEngine** — Delta and period-preserving engines have different correctness trade-offs (delta is production-safe; period-preserving is experimental). Dispatching at service level lets the charge carry its own engine preference without the caller knowing internal engine details.
- **Lazy DetailedLines loading via interface rather than pre-loading in callers** — Prior runs usually have DetailedLines already expanded by the time rating is called; the fetcher interface avoids redundant DB round-trips while providing a safe fallback for cases where they are missing.

## Example: Rate a usage-based charge and retrieve detailed lines with the stored-at cutoff

```
import (
    usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
    billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
)

svc, err := usagebasedrating.New(usagebasedrating.Config{
    StreamingConnector:   streamingConnector,
    RatingService:        billingratingservice.New(),
    DetailedLinesFetcher: detailedLinesFetcher,
})
if err != nil { return err }

result, err := svc.GetDetailedRatingForUsage(ctx, usagebasedrating.GetDetailedRatingForUsageInput{
    Charge:          charge,          // must have State.RatingEngine set
    ServicePeriodTo: currentPeriodTo, // must be within Charge.Intent.ServicePeriod
// ...
```

<!-- archie:ai-end -->
