# Real-time engine (RTE)

RTE runs the existing credit and metered entitlement calculators in a separate worker. It keeps
their last successful results in PostgreSQL and can refresh individual balances when their inputs
change. Product APIs, billing and notifications still use their existing calculation paths.

A credit balance is keyed by customer, currency and feature scope within a namespace. A metered
balance is keyed by namespace and entitlement ID. The domain engines own the amounts, usage
periods and grant rules. See [credit balance keys](creditbalance/README.md) and
[metered entitlement balances](meteredentitlement/README.md).

## Choosing an operation

A caller outside the worker uses `rte.Client`. The caller authorizes access to the namespace and
key before using the client. Both client interfaces expose `CreditBalances()` and
`MeteredEntitlements()`. [Go interfaces](client.go)

| Client operation | What success means |
| --- | --- |
| `Get` | PostgreSQL returned the last stored calculation, even if newer work is pending. A missing result returns a not-found error. A stored zero is a valid balance. |
| `RequestCalculation` | The caller published a request to run the calculator once. An immediate `Get` can still return the previous result. |
| `RequestRefresh` | The caller published a freshness request. The worker may satisfy it with a clean stored result. |
| `Activate` | The key is registered for background refresh. The first result may arrive later. |
| `Deactivate` | Background and temporary tracking end for this key. Its stored result remains readable. |

The two requests go through Kafka. Reads and tracking changes use PostgreSQL. Publish a request
after commit if it depends on a write in the caller's transaction. Retrying a request after a
lost acknowledgement can submit it twice.

Inside a process that hosts the engine, `rte.SyncClient` exposes the same typed accessors.
Use `Calculate` to run an unconditional calculation and return the retained state. Use
`EnsureFresh` to wait for a required refresh, then read the result with `Get` on the ordinary
client. Billing that needs to see its own uncommitted writes calls the domain service directly.

## How a tracked balance stays current

1. A committed source event or stored usage batch identifies balances whose inputs may have
   changed. The worker matches those keys against active demand and marks them dirty in PostgreSQL.
   RTE records the Kafka partition offset in the same write, so replaying that offset skips the invalidation. [Source routing](upstream/README.md)
2. For an activated key, a worker polls for dirty demand and claims the key. For a key tracked
   only by `EnsureFresh`, calculation waits for the next freshness request. Several changes can
   share a calculation. Changes received during the calculation leave another pass pending.
   [Claims, ordering and retries](demand/README.md)
3. On an accepted result, the demand transaction commits the stored balance, its update event
   in the outbox, and completion of the claimed version together. Kafka delivery follows commit.
   A failed calculation leaves the previous value available. [Storage](adapter/README.md) ·
   [Update delivery](../watermill/outbox/README.md)

`PullTTL` controls how long RTE tracks changes between freshness requests. If tracking lapses,
the next request recalculates. `MaxAge` limits how old a clean result the request will accept.
With `MaxAge=0`, a continuously tracked key can look clean indefinitely if a writer misses an
event or time alone changes the balance. A finite maximum age bounds reuse on the next
freshness request. Plain reads and the passage of time do not schedule that refresh.

After a worker outage, retained Kafka messages can still invalidate their keys when consumption
resumes. If a change was never published, replay cannot recover it. A one-time calculation for
an affected key rebuilds its current value from the domain sources.

## Calculation events

Both balance kinds use `io.openmeter.rte.v1.calculationRequested` on the system event topic.
The following CloudEvent data requests a customer's whole USD portfolio, tracks changes for
60 seconds and accepts a clean calculation up to 10 seconds old.

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

Omitting `freshness` requests an unconditional calculation on each delivery. An accepted write
emits `io.openmeter.rte.v1.calculationUpdated` with `kind`, `key` and the complete stored
`state` (`calculated_at` and `value`), even when the amount stays the same. A clean refresh or a
superseded calculation emits no update. Consumers can keep the greatest `calculated_at` per key
to handle duplicate or out-of-order delivery. [Event types](event.go)

## Comparing performance on live traffic

Enable `rte.enabled` on the RTE worker, API, balance worker, and processes writing charges or
ledger postings. The existing path serves the response or notification. Producers sample
successful current operations by balance key (10% by default) so repeat reads of a sampled key
reach the same comparison. Source changes are published independently of that sample.
[Worker configuration](../../cmd/rte/README.md)

| Observed operation | What the worker times |
| --- | --- |
| Current full credit balance or current metered entitlement value | `EnsureFresh` with one minute of tracking and a 30-second maximum age, followed by `Get` |
| Metered recalculation selected by the balance worker | `Calculate`, including storage and enqueueing an accepted update |

The comparison records paired durations for successful replays. It discards the returned value.
The delay before replay, asynchronous event delivery and product HTTP latency fall outside those durations.
Check drops, replay lag and database load alongside the paired timings. To end the trial, disable
the flag and restart the participating processes. [Metrics and example query](metrics/README.md#reading-the-performance-comparison)

## Adding a calculation

Define the typed key and result, adapt the domain calculator, and pair it with a transactional store
in `service.Registration`. Expose the kind through the service and client accessors. Give it an event
kind and key decoder, and map the source events that can change it to demand selectors.
Test a matching key and an unaffected key through the calculator and store.
[Calculation service](service/README.md) · [RTE tests](../../test/rte/README.md)
