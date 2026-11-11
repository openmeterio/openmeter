# Real-time engine (RTE)

RTE maintains current credit and metered entitlement balances for callers that ask for them.
The ledger and entitlement engines still define the answers. RTE decides when to run those
engines, stores their last successful result in PostgreSQL, and publishes accepted updates.
The existing product balance APIs and notification worker continue to use their own paths.

## Choosing an operation

| Caller needs | Operation | What happens |
| --- | --- | --- |
| The last stored value | `Get(key)` | Reads state even if demand is dirty or expired. A missing row returns a not-calculated error. A stored zero is a successful result. |
| An exact calculation now | `Calculate(key)` | Always runs the domain calculator and returns the state retained by the store. Creates no ongoing interest. |
| A current value on demand | `EnsureFresh(target, …)`, then `Get(key)` | Registers or extends pull interest. Waits for any necessary calculation, then the caller can read state. Repeated clean pulls can reuse the result. |
| Background updates | `Activate(target, …)` | Registers persistent push interest. A worker calculates an initially dirty key and processes subsequent changes. Activation may return before the first result exists. |
| To stop tracking a key | `Deactivate(target)` | Clears both push and pull interest for the target. The last stored value stays readable. |

Use the typed accessors on `rte.Service` for `Get` and `Calculate`. For the other operations,
construct a target with `rte.NewCalculationTarget(key)` and pass it to `rte.DemandService`.
A positive pull TTL controls how long the worker tracks input changes between reads. When
that interest lapses and nothing else keeps the key active, the next pull calculates again.
`MaxAge` limits how old a completed calculation the caller accepts. Zero accepts any age
while the tracked key is clean. A push `RefreshInterval` schedules periodic work even when
no event arrives. Zero leaves refreshes event-driven.

## What identifies a balance

| Kind | Key | Stored value |
| --- | --- | --- |
| `credit_balance` | Namespace, customer ID, fiat or custom currency identity, and feature scope | Booked, spendable, live and pending credit from the current [customer balance calculation](../ledger/customerbalance/README.md). |
| `metered_entitlement` | Namespace and entitlement ID | The metered engine's `EntitlementBalance`, including remaining balance, available grant amount and usage in the period. |

Credit feature scopes have distinct wire representations. Missing or `null` selects the whole
portfolio, `[]` selects unrestricted credits, and a single feature selects credits eligible
for that feature. Mixing these up changes both the value and which key receives updates.
Metered calculations pass the RTE start time to the entitlement engine, which owns minute
rounding, grant eligibility and source snapshots. Recalculating late usage therefore inherits
that engine's behavior for usage earlier than an existing snapshot.

## From input change to stored update

1. **Find interested keys.** The [upstream resolver](upstream/README.md) maps a committed fact or stored
   usage batch to the affected targets. Usage routing considers subject attribution, meters,
   charge service periods and credit scope. A fact for a customer without active demand creates
   no background work.
2. **Mark those keys dirty.** RTE records the source and event ID and advances the dirty version
   for matching active keys in one transaction. Redelivery within the receipt retention window
   has no further effect. A distinct event for earlier usage can still advance that version.
3. **Calculate once for pending changes.** A worker claims a ready push key. Pull-only keys wait
   for the next `EnsureFresh`. Multiple input changes before the claim can share one calculation.
   An invalidation racing with a claimed key waits for the lock and can schedule another pass.
4. **Store and announce the accepted result.** A successful demand transaction commits state,
   its update outbox entry and demand completion together. Delivery to Kafka happens later.
   A failed demand calculation leaves the previous value available. The worker retries push
   work with backoff and a pull can retry immediately. See [demand processing](demand/README.md)
   and [outbox delivery](../watermill/outbox/README.md).

The worker subscribes to separate ingest and system event topics. It handles customer,
subscription, feature, meter, entitlement, grant and reset facts. Enable `rte.publishSourceEvents`
on **billing and ledger writers** to cover charge creation, charge patches and committed ledger
posting groups. For example, a pending credit purchase can affect the answer before it produces
any postings. When an ingest batch affects credits, the worker waits until the full stored
second is visible to the ClickHouse rating query before invalidating those keys.

Freshness is relative to the source facts RTE has processed. Set a finite `MaxAge` or
`RefreshInterval` when the answer can change with time alone or through a writer without a
matching source event, including charge lifecycle writes outside creation and patching.

### Request and update contracts

On the system event topic, both calculators accept `io.openmeter.rte.v1.calculationRequested`.
The `kind` selects the schema of `key`. For example, this CloudEvent `data` payload requests
a customer's whole USD portfolio, keeps pull interest for one minute, and accepts a result
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

Omitting `freshness` runs an explicit calculation on each delivery. A clean pull emits no
update, so a caller that needs a response should use `EnsureFresh` followed by `Get`.

On an accepted write, `io.openmeter.rte.v1.calculationUpdated` carries `kind`, `key`, and
the complete persisted `state` with `calculated_at` and `value`. An accepted recalculation emits an update even if its numeric
result is unchanged. If a slower calculation finishes after one with an equal or later start,
the store keeps the existing state and the slower attempt returns that state without publishing.
Consumers can keep the greatest `calculated_at` per target to tolerate duplicate or out-of-order
updates. This timestamp orders calculation starts across replicas and requires aligned clocks.
Each source follows its own read and snapshot semantics.

### Example with both balances

Consider a customer with 100 credits and an entitlement with a 100-unit grant. Both keys have
push demand. In the [live worker test](../../test/rte/live_process_test.go), a clean credit pull
reuses the initial result. Adding 25 credits through the ledger raises live credit to 125 and
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
- Give the kind a wire name and wire up key decoding, semantic identity, event metadata,
  demand validation, and request and demand dispatch. The available kinds are explicit in code.
  Extend [upstream routing](upstream/README.md) for the facts that can change the result. Document which
  changes callers must cover with a refresh interval or maximum age.
- Exercise the real calculator and store and check both an affected key and an unaffected key.
  The [RTE test guide](../../test/rte/README.md) covers the PostgreSQL integration suite and the
  separate worker process running against Kafka and ClickHouse.

## Running and observing RTE

`make build-rte` builds `build/rte`. Run `build/rte --config config.yaml` with `credits.enabled`
and the usual event, storage and telemetry settings. See the [worker command](../../cmd/rte/README.md). `rte.consumerGroupName`, `rte.retry` and
`rte.dlq` configure message consumption. `rte.pollInterval` controls idle polling for demand
(default 1s) and `rte.calculationTimeout` bounds a demand attempt (default 30s). Subscription
health is available at `/healthz/subscriptions` on the telemetry server.

| Operational question | OpenTelemetry instruments |
| --- | --- |
| How often did each engine run, and how long did it take to calculate and store? | `rte.calculations` and `rte.calculation.duration` by `kind` and `outcome` (`updated`, `superseded`, `failed`). |
| Did a pull reuse state? How many tracked keys did incoming changes dirty? | `rte.demand.operations` by operation and outcome, and `rte.demand.invalidated` for the matched key count. |
| Are input facts arriving late or missing relevant targets? | `rte.upstream.lag` measures time from publication to handling. `rte.upstream.events` distinguishes `routed`, `no_target` and `error`. |
| Are direct reads finding state, and is rating work being reused? | `rte.state.reads` reports hits, misses (`missing`) and failures. `rte.rating.lookups` counts per-charge hits and misses. |
