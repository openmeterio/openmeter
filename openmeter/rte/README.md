# Real-time engine (RTE)

RTE maintains current credit and metered entitlement balances for callers that ask for them.
The ledger and entitlement engines still define the answers. RTE decides when to run those
engines, stores their last successful result in PostgreSQL, and publishes accepted updates.
The existing product balance APIs and notification worker continue to use their own paths.

## Choosing an operation

| Caller needs | Operation | What happens |
| --- | --- | --- |
| The last stored value | `Get(key)` | Reads state even if demand is dirty or expired. A missing row returns a not-calculated error. A stored zero is a successful result. |
| A one-time calculation | `Calculate(key)` | Runs the domain calculator once and returns the stored result without starting change tracking. |
| A current value on demand | `EnsureFresh(target, …)`, then `Get(key)` | Tracks changes between requests. Reuses a clean result within the caller’s age limit or waits for a recalculation. |
| Background updates | `Activate(target, …)` | Enables background refresh. A worker calculates the initial balance and refreshes it after tracked changes. Activation can return before the first result is ready. |
| To stop tracking a key | `Deactivate(target)` | Stops background refresh and change tracking for the key. The last stored value stays readable. |

Use the typed accessors on `rte.Service` for `Get` and `Calculate`. For the other operations,
construct a target with `rte.NewCalculationTarget(key)` and pass it to `rte.DemandService`.
Each `EnsureFresh` call extends change tracking for at least `PullTTL`. If tracking expires and
background refresh is off, the next `EnsureFresh` recalculates to cover any missed changes.
`MaxAge` limits how old a completed calculation the caller accepts. Zero accepts any age
while the tracked key is clean. With background refresh enabled, `RefreshInterval` schedules
periodic work even when no event arrives. Set it to zero to disable periodic refreshes.

## What identifies a balance

| Kind | Key | Stored value |
| --- | --- | --- |
| `credit_balance` | Namespace, customer ID, fiat or custom currency identity, and feature scope | Booked, spendable, live and pending credit from the current [customer balance calculation](../ledger/customerbalance/README.md). |
| `metered_entitlement` | Namespace and entitlement ID | The metered engine's `EntitlementBalance`, including remaining balance, available grant amount and usage in the period. |

In a JSON request, omit `featureFilter` or set it to `null` for the whole credit portfolio.
Use `[]` for unrestricted credits only. Use `["search"]` for unrestricted credits plus credits
eligible for `search`. RTE tracks each of these balances separately.
Metered calculations pass the RTE start time to the entitlement engine, which owns minute
rounding, grant eligibility and source snapshots. Recalculating late usage therefore inherits
that engine's behavior for usage earlier than an existing snapshot.

## From input change to stored update

1. **Find affected keys.** The [upstream resolver](upstream/README.md) maps a committed fact or stored
   usage batch to the affected targets. Usage routing considers subject attribution, meters,
   charge service periods and credit scope. A fact for a customer without active demand creates
   no background work.
2. **Mark those keys dirty.** RTE records the source and event ID and advances the dirty version
   for matching active keys in one transaction. Redelivery within the receipt retention window
   has no further effect. A distinct event for earlier usage can still advance that version.
3. **Calculate once for pending changes.** With background refresh enabled, a worker claims the key.
   Otherwise, recalculation waits for the next `EnsureFresh`. One calculation can cover several
   input changes.
   An invalidation racing with a claimed key waits for the lock and can schedule another pass.
4. **Store and announce the accepted result.** A successful demand transaction commits state,
   its update outbox entry and demand completion together. Delivery to Kafka happens later.
   A failed demand calculation leaves the previous value available. Background refreshes retry
   with backoff. An explicit `EnsureFresh` can retry immediately. See [demand processing](demand/README.md)
   and [outbox delivery](../watermill/outbox/README.md).

The worker subscribes to separate ingest and system event topics. It handles customer,
subscription, feature, meter, entitlement, grant and reset facts. Enable `rte.enabled`
on **billing and ledger writers** to cover charge creation, charge patches and committed ledger
posting groups. For example, a pending credit purchase can affect the answer before it produces
any postings. For an ingest batch, the worker resolves demand after the batch’s storage second
ends. This includes credit keys whose first calculation ran too early to count that batch.

Freshness is relative to the source facts RTE has processed. Set a finite `MaxAge` or
`RefreshInterval` when the answer can change with time alone or through a writer without a
matching source event, including charge lifecycle writes outside creation and patching.

### Request and update contracts

On the system event topic, both calculators accept `io.openmeter.rte.v1.calculationRequested`.
The `kind` selects the schema of `key`. For example, this CloudEvent `data` payload requests
a customer's whole USD portfolio, tracks changes for one minute, and accepts a result
up to ten seconds old.

```json
{
  "kind": "credit_balance",
  "key": {
    "customerId": {"namespace": "example", "id": "customer-1"},
    "currency": {"code": "USD"},
    "featureFilter": null
  },
  "freshness": {"track_for_seconds": 60, "max_age_seconds": 10}
}
```

Omitting `freshness` runs an explicit calculation on each delivery. Reusing a clean result emits no
update, so a caller that needs a response should use `EnsureFresh` followed by `Get`.

On an accepted write, `io.openmeter.rte.v1.calculationUpdated` carries `kind`, `key`, and
the complete persisted `state` with `calculated_at` and `value`. An accepted recalculation emits an update even if its numeric
result is unchanged. If a slower calculation finishes after one with an equal or later start,
the store keeps the existing state and the slower attempt returns that state without publishing.
Consumers can keep the greatest `calculated_at` per target to tolerate duplicate or out-of-order
updates. This timestamp orders calculation starts across replicas and requires aligned clocks.
Each source follows its own read and snapshot semantics.

### Example with both balances

Consider a customer with 100 credits and an entitlement with a 100-unit grant. Background refresh is enabled
for both keys. In the [live worker test](../../test/rte/live_process_test.go), `EnsureFresh`
reuses the initial credit balance. Adding 25 credits through the ledger raises live credit to 125 and
leaves the entitlement untouched. A late batch of 25 units priced at one credit per unit then
refreshes both keys. Booked credit stays at 125, live credit falls to 100, and the entitlement
has 75 units left. Redelivering that batch leaves both timestamps and execution counts unchanged.

The funding change illustrates two levels of reuse. Target selection skips the entitlement
calculation entirely. Within the credit calculation, rating still reads fresh usage and can
reuse the previous per-charge total when all rating inputs match. This bounded [rating cache](../billing/charges/usagebased/service/rating/README.md) belongs
to the credit calculator process. A restart loses cache entries and preserves stored balances.

## Adding a calculation

- Define a validated current-state key and the complete result in a domain package. Implement
  `rte.Calculator[K, Result]` in its calculator subpackage, using the domain's existing source
  of truth. See the [credit](creditbalance/README.md) and [metered entitlement](meteredentitlement/README.md) implementations.
- Supply a typed `rte.Store[K, Result]` with [transactional writes](adapter/README.md). Match persisted identity to
  demand identity, apply the strictly-later-start rule and return the actual retained state.
  Pair calculator and store in `service.Registration` and expose the typed accessor through
  `rte.Service`. The [shared service](service/README.md) owns update publication for each registration.
- Choose the `kind` value that identifies the calculation in events. Use it when decoding
  keys, identifying balances, building event metadata, validating demand and dispatching work.
  Extend [upstream routing](upstream/README.md) for the facts that can change the result. Document which
  changes callers must cover with a refresh interval or maximum age.
- Exercise the real calculator and store and check both an affected key and an unaffected key.
  The [RTE test guide](../../test/rte/README.md) covers the PostgreSQL integration suite and the
  separate worker process running against Kafka and ClickHouse.

## Running and observing RTE

`make build-rte` builds `build/rte`. Run `build/rte --config config.yaml` with `rte.enabled`, `credits.enabled`
and the usual event, storage and telemetry settings. See the [worker command](../../cmd/rte/README.md). `rte.consumerGroupName`, `rte.retry` and
`rte.dlq` configure message consumption. `rte.pollInterval` controls idle polling for demand
(default 1s) and `rte.calculationTimeout` bounds a demand attempt (default 30s). Subscription
health is available at `/healthz/subscriptions` on the telemetry server.

| Operational question | OpenTelemetry instruments |
| --- | --- |
| How often did each engine run, and how long did it take to calculate and store? | `rte.calculations` and `rte.calculation.duration` by `kind` and `outcome` (`updated`, `superseded`, `failed`). |
| Did `EnsureFresh` reuse the stored result? How many tracked keys did incoming changes dirty? | `rte.demand.operations` by operation and outcome, and `rte.demand.invalidated` for the matched key count. |
| Are input facts arriving late or missing relevant targets? | `rte.upstream.lag` measures time from publication to handling. `rte.upstream.events` distinguishes `routed`, `no_target` and `error`. |
| Are direct reads finding state, and is rating work being reused? | `rte.state.reads` reports hits, misses (`missing`) and failures. `rte.rating.lookups` counts per-charge hits and misses. |

### Comparing performance on live traffic

Set `rte.enabled: true` on the API, balance worker, billing and ledger writers, and RTE worker.
The default sample is 10% of balance keys. Use the same `rte.shadowSamplePercent` on each producer.
Sampling by key preserves repeat reads. Committed source changes and the sink’s existing
stored-usage notifications invalidate tracked balances across the sample.

| Production work observed | Operation timed on the RTE worker |
| --- | --- |
| Current full credit balance, per resolved currency and feature scope | `EnsureFresh` then `Get` |
| Current metered entitlement value, after entitlement lookup | `EnsureFresh` then `Get` |
| Current metered recalculation selected by the existing balance worker | `Calculate`, including storage and update publication to the outbox |

The existing path serves every response and notification. On successful operations a bounded
queue hands a copy of the request to Kafka. Historical and booked-only reads are excluded.
A shadow read tracks changes for one minute and accepts a clean result up to 30 seconds old.
The metered calculator can read existing entitlement snapshots. Only the production reader
maintains them. Shadow calculations write RTE state and the usual RTE update events.

**Reading the measurements**

- Compare `rte.comparison.duration` (ms) by `kind`, `operation` and `variant` (`direct` or `rte`).
  Each successful replay contributes one measurement per variant. Divide each duration sum by
  its count for the mean, or compare p95s across the same window. To weight by real traffic,
  sum the durations across operations before dividing by the total count.
- Check `rte.comparison.publications` on producers for drops or failed deliveries,
  `rte.comparison.executions` on the worker for failures and expired samples, and
  `rte.comparison.lag` for time between the original call and replay. An observation expires
  after one minute. Kafka redelivery can repeat an observation. Lost source facts or a backlog can inflate the apparent cache hit rate.
- Use `rte.demand.ensure_fresh.duration` (s) by `kind` and `outcome` to separate cache hits
  from recalculations. `rte.state.read.duration` and `rte.calculation.duration` (s) show the
  state read and calculation costs. These timings overlap the paired RTE measurement.
- The comparison measures balance work once the key is known. HTTP handling, currency or
  entitlement lookup, Kafka lag, and asynchronous outbox delivery fall outside that timing.
  `rte.upstream.routing.duration` (s) measures the additional work to resolve source events
  and mark balances dirty, including events that match no target.
  Check API latency, database and ClickHouse load alongside it. Shadow traffic adds real work
  and shares source data and database caches with the existing path.

For the percentage of balance-operation time saved in the observed read/calculation mix,
use the paired histogram sums. A negative value means RTE took longer.

```promql
100 * (1 -
  sum by (kind) (rate(rte_comparison_duration_milliseconds_sum{variant="rte"}[1h]))
  / sum by (kind) (rate(rte_comparison_duration_milliseconds_sum{variant="direct"}[1h]))
)
```

Disable `rte.enabled` and restart or redeploy the participating processes to end the trial.
