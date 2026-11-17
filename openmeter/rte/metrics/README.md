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
| `rte.state.reads` | Counter | `kind`, `outcome` | Calculation-service `Get` calls: `hit`, `missing` (not calculated), or `failed`. |
| `rte.state.read.duration` | Histogram / s | `kind`, `outcome` | Time spent reading stored state, without refreshing it. |
| `rte.demand.operations` | Counter | `operation`, `outcome` | Freshness requests, invalidations, processing attempts and receipt maintenance; see outcomes below. |
| `rte.demand.ensure_fresh.duration` | Histogram / s | `kind`, `outcome` | `EnsureFresh`, including demand registration, lock waits and any required calculation. Outcomes: `cache_hit`, `calculated`, `error`. |
| `rte.demand.invalidated` | Counter | None | Sum of active keys matched by invalidations. Counts matches, not distinct keys or eventual calculations. |
| `rte.upstream.events` | Counter | `event`, `outcome` | Supported source-event handling: `routed`, `no_target` or `error`. `event` is the event type, never its ID. |
| `rte.upstream.lag` | Histogram / s | `event` | Time from the CloudEvent's publication timestamp to handling, when that timestamp exists. Does not measure the age of the underlying usage. |
| `rte.upstream.routing.duration` | Histogram / s | `event`, `outcome` | Target resolution and invalidation, after any stored-usage visibility wait. Includes routing that finds no target. |
| `rte.rating.lookups` | Counter | `outcome` | Per-charge rating reuse: `hit` or `miss`. Fresh usage is still queried before lookup; a hit only avoids rating work. |
| `rte.comparison.publications` | Counter | `event`, `kind`, `operation`, `outcome` | Producer-side observation/source publication outcomes. `event` is `observation` or `source`; source facts have empty `kind` and `operation`. |
| `rte.comparison.executions` | Counter | `kind`, `operation`, `outcome` | Worker-side shadow replays: `success`, `error` or `expired`. |
| `rte.comparison.duration` | Histogram / ms | `kind`, `operation`, `variant` | Paired successful production and RTE timings, with `variant=direct` and `variant=rte`. |
| `rte.comparison.lag` | Histogram / ms | `kind`, `operation` | Delay from completion of the production observation to worker handling, including observations that have expired. |

Calculation and stored-read instruments cover the calculation service, including
shadow replays. The ordinary client's direct-database `Get` bypasses these
instruments. Validation failures before a timed operation starts are also excluded.
Calculation outcomes do not prove that an enclosing demand transaction committed
or that Kafka delivered the update.

## Demand and publication outcomes

| `rte.demand.operations` operation | Outcomes |
| --- | --- |
| `pull` | `cache_hit`: no calculation needed; `calculated`: this call processed the refresh; `error`: registration or refresh failed. |
| `invalidate` | `marked`: accepted nonduplicate invalidation, possibly matching zero keys; `duplicate`: receipt already exists; `error`: invalidation failed. |
| `process` | `empty`: no work needed or claimable; `calculated`: processed successfully; `retired`: source deletion ended tracking; `retry`: calculation failed and was deferred; `error`: processing failed. Includes background attempts and processing invoked by `EnsureFresh`. |
| `maintenance` | `pruned`: removed expired receipts; `empty`: none removed; `error`: pruning failed. Counts executed maintenance passes, not every poll. |

`Activate` and `Deactivate` do not currently increment this counter. A pull can
also increment `process`; do not sum the two to count requests. `routed` upstream
events can still produce a duplicate invalidation or match no active demand.
Several dirty events can share one calculation.

Publication outcomes are `published`, `failed`, `queue_full`, `stopped`, `expired`,
`sampled_out` and `invalid`. `published` means the publisher returned successfully,
not that the worker processed the message. Only observations are sampled or expire;
source facts are eligible regardless of sampling. Sampling exclusions are expected,
while queue drops and failed delivery reduce coverage.

## Reading the performance comparison

See [live-trial setup](../README.md#comparing-performance-on-live-traffic) for
sampling and activation. A shadow `read` times `EnsureFresh + Get`; `calculate`
times an explicit calculation including storage and update enqueueing.

- Compare `rte.comparison.duration` by the same `kind`, `operation` and time window.
  Each successful replay records one sample per variant. Failed or expired replays
  contribute no pair. Use histogram sum/count for means; compare p95s per operation.
- Check producer publications, worker execution outcomes and replay lag alongside
  timings. Observations expire after one minute; Kafka redelivery can repeat a
  sample. Lost source facts or a backlog can inflate apparent cache reuse.
- Split `rte.demand.ensure_fresh.duration` by outcome to separate reuse from refresh
  costs. Calculation and state-read durations explain parts of the paired RTE
  measurement; they overlap it and must not be added to it.
- These timings begin after the balance key is known. HTTP handling, currency or
  entitlement lookup, Kafka lag and asynchronous outbox delivery are outside the
  comparison. Source routing is additional work measured separately. Check API
  latency and database/ClickHouse load too: shadow traffic adds work and shares
  caches with the existing path.

For time saved across the observed sampled operation mix, compare the paired
histogram sums. A negative value means RTE took longer. This is not total-system
savings or a forecast for a different read/calculation mix.

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

Calculation, stored-read, freshness and routing histograms share explicit buckets
from 0.5 ms to 30 s, recorded in seconds. Comparison durations use 0.1–30,000 ms;
comparison lag uses 1–300,000 ms. Upstream lag uses the meter provider's default
boundaries. Bucket limits do not limit recorded values or operation runtimes.
