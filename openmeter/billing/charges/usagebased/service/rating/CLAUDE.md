# rating

<!-- archie:ai-start -->

> Stateless computation sub-package that snapshots metered usage from ClickHouse and converts it into rated detailed lines or totals via the parent billing rating service. No DB writes — all persistence is handled by callers.

## Patterns

**Config-struct constructor with Validate()** — New(Config) validates all required fields before returning a Service. Every exported input type also implements Validate(). (`func New(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**StoredAtOffset cutoff in meter queries** — All ClickHouse queries use FilterStoredAt.Lt = &StoredAtOffset so usage is bounded to a deterministic point in time, enabling idempotent re-rating. (`FilterStoredAt: &filter.FilterTimeUnix{FilterTime: filter.FilterTime{Lt: &in.StoredAtOffset}}`)
**Service-period-scoped ChildUniqueReferenceID** — After rating, every DetailedLine.ChildUniqueReferenceID is suffixed with the UTC service period via withServicePeriodInDetailedLineChildUniqueReferenceIDs to guarantee global uniqueness across reruns. (`"unit-price-usage@[2025-01-01T00:00:00Z..2025-02-01T00:00:00Z]"`)
**Prefer GetTotalsForUsage over GetDetailedLinesForUsage when only totals needed** — GetTotalsForUsage skips detailed-line construction, making it faster. Both methods share the same snapshotQuantity helper. (`totals, err := svc.GetTotalsForUsage(ctx, GetTotalsForUsageInput{...})`)
**PriorRuns must have DetailedLines expanded before rating** — GetDetailedLinesForUsageInput.Validate() checks that all PriorRuns have DetailedLines.IsPresent(); callers must pre-fetch via adapter.FetchDetailedLines. (`if !run.DetailedLines.IsPresent() { return fmt.Errorf("prior runs[%d]: detailed lines must be expanded") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the Service interface (GetTotalsForUsage, GetDetailedLinesForUsage), Config struct, and New constructor. | This is a pure computation service — no Ent/DB dependency; never add adapter or tx logic here. |
| `details.go` | Implements GetDetailedLinesForUsage: snapshots quantity, calls ratingService.GenerateDetailedLines, rewrites ChildUniqueReferenceIDs. | IgnoreMinimumCommitment flag must be threaded as a billingrating.WithMinimumCommitmentIgnored() option — not a conditional branch. |
| `totals.go` | Implements GetTotalsForUsage: same snapshot + rating path but extracts only ratingResult.Totals. | Does NOT suffix ChildUniqueReferenceIDs — do not add that logic here. |
| `quantitysnapshot.go` | Private snapshotQuantity helper that queries ClickHouse via streaming.Connector.QueryMeter and sums all rows. | Validation error wrapped as billing.ValidationError{Err: err} — use the same pattern for new validations. |
| `uniqueref.go` | formatDetailedLineChildUniqueReferenceID and withServicePeriodInDetailedLineChildUniqueReferenceIDs — pure string transformations. | Timestamps formatted with time.RFC3339 in UTC; changing format breaks deduplication of existing persisted lines. |
| `service_test.go` | Unit tests using streamingtestutils.NewMockStreamingConnector and a stub ratingService — no DB required. | Tests use t.Context(); never substitute context.Background(). |

## Anti-Patterns

- Adding Ent/DB adapter calls inside this package — all persistence is handled by callers in the run package.
- Calling streaming.Connector with a nil StoredAtOffset — every query must respect the stored-at cutoff.
- Skipping ChildUniqueReferenceID suffixing when adding new detailed-line paths — persisted lines will collide across reruns.
- Returning mutable slices from snapshotQuantity without copying — summarizeMeterQueryRow already returns a new value, keep it immutable.
- Bypassing Config.Validate() or input.Validate() before use — all constructors/methods must validate first.

## Decisions

- **Stateless package with no DB dependency** — Rating is a pure computation: snapshot usage + apply rate card. Keeping it DB-free makes it trivially testable with mock connectors and reusable from multiple callers without transaction concerns.
- **Service-period suffix on ChildUniqueReferenceID** — The same rate-card key (e.g. 'unit-price-usage') is reused across multiple billing periods. Appending the UTC period in RFC3339 makes each child reference globally unique so upsert logic in the run package can safely idempotently re-persist lines.

## Example: Rate a usage-based charge and retrieve detailed lines with the stored-at cutoff

```
import (
    usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
    billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
    "github.com/openmeterio/openmeter/openmeter/streaming"
)

svc, err := usagebasedrating.New(usagebasedrating.Config{
    StreamingConnector: streamingConnector,
    RatingService:      ratingService,
})
if err != nil { return err }

result, err := svc.GetDetailedLinesForUsage(ctx, usagebasedrating.GetDetailedLinesForUsageInput{
    Charge:          charge,
    PriorRuns:       charge.Realizations, // must have DetailedLines expanded
// ...
```

<!-- archie:ai-end -->
