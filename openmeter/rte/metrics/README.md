# RTE metrics

This package owns instrument construction, histogram boundaries, labels and
recording methods. Callers decide when an operation starts and which outcome it
reaches. Names below are OpenTelemetry instrument names; Prometheus converts dots
to underscores and adds counter/unit/histogram suffixes.

## Instruments

`kind` is `credit_balance` or `metered_entitlement`. Comparison `operation` is
`read` or `calculate`. Durations are wall-clock elapsed time, including waits
inside the measured operation.

| Instrument | Type / unit | Labels | What it measures |
| --- | --- | --- | --- |
| `rte.calculations` | Counter | `kind`, `outcome` | Calculation calls: `updated` when the store accepts the result, `superseded` when it retains a newer or equal-start result, `failed` on error. |
| `rte.calculation.duration` | Histogram / s | `kind`, `outcome` | The same calls, including domain calculation, state storage and update enqueueing. Excludes asynchronous outbox delivery. |
| `rte.state.reads` | Counter | `kind`, `outcome` | Client and calculation-service `Get` calls: `hit`, `missing` (not calculated), or `failed`. |
| `rte.state.read.duration` | Histogram / s | `kind`, `outcome` | Time spent reading stored state, without refreshing it. |
| `rte.demand.operations` | Counter | `operation`, `outcome` | Freshness requests, invalidations and processing attempts. |
| `rte.demand.ensure_fresh.duration` | Histogram / s | `kind`, `outcome` | `EnsureFresh`, including demand registration, lock waits and any required calculation. Outcomes: `cache_hit`, `calculated`, `error`. |
| `rte.demand.queue_wait.duration` | Histogram / s | `kind`, `outcome` | Time from eligibility to claim. For another pass on the same key, eligibility starts when the previous pass finishes. A retry becomes eligible at its backoff deadline. |
| `rte.demand.processing.duration` | Histogram / s | `kind`, `outcome` | Elapsed time from the confirmed background claim through processing and transaction completion. Both background histograms record when the attempt returns. Empty polls and targeted pulls contribute no samples. |
| `rte.demand.invalidated` | Counter | None | Sum of active keys matched by invalidations. Counts matches, not distinct keys or eventual calculations. |
| `rte.upstream.events` | Counter | `event`, `outcome` | Supported source-event handling: `routed`, `no_target` or `error`. `event` is the event type, never its ID. |
| `rte.upstream.lag` | Histogram / s | `event` | Time from the CloudEvent's publication timestamp to handling, when that timestamp exists. Does not measure the age of the underlying usage. |
| `rte.upstream.routing.duration` | Histogram / s | `event`, `outcome` | Target resolution and invalidation, after any stored-usage visibility wait. Includes routing that finds no target. |
| `rte.rating.lookups` | Counter | `outcome` | With `rte.reuseRatingTotals` enabled, per-charge rating reuse: `hit` or `miss`. Fresh usage is still queried before lookup; a hit only avoids rating work. |
| `rte.comparison.publications` | Counter | `event`, `kind`, `operation`, `outcome` | Producer-side observation/source publication outcomes. `event` is `observation` or `source`; source facts have empty `kind` and `operation`. |
| `rte.comparison.executions` | Counter | `kind`, `operation`, `outcome` | Worker-side shadow replays: `success`, `error` or `expired`. |
| `rte.comparison.duration` | Histogram / ms | `kind`, `operation`, `variant` | Paired successful production and RTE timings, with `variant=direct` and `variant=rte`. |
| `rte.comparison.lag` | Histogram / ms | `kind`, `operation` | Delay from completion of the production observation to worker handling, including observations that have expired. |

Calculation instruments cover the calculation service, including shadow replays.
Stored-read instruments also cover the ordinary client's direct-database `Get`,
using the meter supplied at client construction. Each path records once; raw
adapter reads are not instrumented. Validation failures before a timed operation
starts are excluded.
Calculation outcomes do not prove that an enclosing demand transaction committed
or that Kafka delivered the update.

## Demand and publication outcomes

| `rte.demand.operations` operation | Outcomes |
| --- | --- |
| `pull` | `cache_hit`: no calculation needed; `calculated`: this call processed the refresh; `error`: registration or refresh failed. |
| `invalidate` | `marked`: applied an invalidation, possibly matching zero keys; `duplicate`: offset already covered; `error`: invalidation failed. |
| `process` | `empty`: no work needed or claimable; `calculated`: processed successfully; `retired`: source deletion ended tracking; `retry`: calculation failed and was deferred; `error`: processing failed. Includes background attempts and processing invoked by `EnsureFresh`. |

`Activate` and `Deactivate` do not currently increment this counter. A pull can
also increment `process`; do not sum the two to count requests. `routed` upstream
events can still match zero active keys.
Several dirty events can share one calculation.

Publication outcomes are `published`, `failed`, `queue_full`, `stopped`, `expired`,
`sampled_out` and `invalid`. `published` means the publisher returned successfully,
not that the worker processed the message. Only observations are sampled or expire;
source facts are eligible regardless of sampling. Sampling exclusions are expected,
while queue drops and failed delivery reduce coverage.

## Reading the performance comparison

Each successful replay records two durations with the same kind and operation. The direct
sample is the original operation's elapsed time. The RTE sample measures the worker's replay.
A read includes `EnsureFresh` and `Get`. A calculation includes storage and enqueueing an
accepted update. The worker discards the value after timing the operation.
[Sampling and trial setup](../README.md#comparing-performance-on-live-traffic)

- Compare the two variants for the same kind, operation and time window. Use histogram sum
  divided by count for means, or compare latency percentiles separately for reads and calculations.
- Check publication drops, replay errors, expiry and lag alongside latency. Failed or expired
  replays contribute no timing pair. Redelivery can repeat a sample. Missed source events can
  make reuse look cheaper because RTE never learned that the inputs changed.
- To explain a slow read, split `rte.demand.ensure_fresh.duration` by outcome. To explain
  delayed background updates, compare `rte.demand.queue_wait.duration` with
  `rte.demand.processing.duration`. Only claimed work contributes to the latter two histograms,
  so an idle or stopped worker produces no evidence about outstanding queue age.
- Product HTTP handling, entitlement or currency lookup, Kafka lag and asynchronous update
  delivery fall outside the paired timings. Check API latency and database and ClickHouse load
  separately. The trial runs both paths and they share infrastructure.

This query gives the percentage reduction in timed work for each kind, weighted by the mix
of successfully replayed operations in the last hour. Negative values mean RTE took longer.

```promql
100 * (1 -
  sum by (kind) (rate(rte_comparison_duration_milliseconds_sum{variant="rte"}[1h]))
  / sum by (kind) (rate(rte_comparison_duration_milliseconds_sum{variant="direct"}[1h]))
)
```

## Maintaining the instruments

Keep labels bounded: calculation kinds, supported event types, operations and
outcomes. Never add customer IDs, namespaces or calculation keys. Instrument
names, units and labels are an observability contract for dashboards and comparisons.

Calculation, stored-read, freshness, background processing and routing histograms share explicit buckets
from 0.5 ms to 30 s, recorded in seconds. Background queue wait uses 1 ms to 1 hour. Comparison durations use 0.1–30,000 ms;
comparison lag uses 1–300,000 ms. Upstream lag uses the meter provider's default
boundaries. Bucket limits do not limit recorded values or operation runtimes.
