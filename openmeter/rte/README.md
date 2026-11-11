# RTE

RTE maintains current credit balances and metered entitlement balances. Each calculator
supplies its key and result. RTE owns stored state, demand, refresh scheduling and update
delivery. The existing balance APIs and notification worker keep their current behavior.

| Operation | When the calculator runs |
| --- | --- |
| `Get` | Reads the last successful result. A missing result returns “not calculated.” |
| `Calculate` | Always runs the exact domain calculator. |
| `EnsureFresh` | Runs on first demand, on invalidation, after pull interest expires, or when the result exceeds the caller's maximum age. Otherwise reuses state. |
| `Activate` | Schedules an initial result and keeps the key fresh in the background. An optional refresh interval also schedules work as time passes. |
| `Deactivate` | Ends tracking for this key. The last result remains readable. |

A credit key includes namespace, customer, currency identity and feature scope (whole
portfolio, unrestricted, or eligible for one feature). Credit reads use the current
`customerbalance` calculation, including booked, spendable, live and pending amounts.
The process can reuse successful per-charge rating results when fresh usage and all
rating inputs match. Metered keys identify an entitlement in a namespace and use the
existing entitlement engine, including its minute rounding, grants and snapshots.

## Events and freshness

The worker consumes ingest and system events and resolves them to active keys.
Usage routing considers subject attribution, meter, service period and credit scope.
For credit targets the worker waits until rating can see the entire stored batch
at ClickHouse’s one-second storage-time precision.
Customer, entitlement, grant, reset, feature, meter and subscription facts can also
invalidate demand. Enable `rte.publishSourceEvents` on billing and ledger writers
to publish charge creations, charge patches and committed posting groups through
the existing outbox. A pending credit purchase can affect a balance before it
has any ledger postings.

Push demand is refreshed by the worker. Pull demand accumulates changes until
the next pull. Multiple changes before a refresh require one exact calculation.
Use a finite refresh interval or maximum age for time boundaries and changes without
a source event, including charge lifecycle writes outside the create/patch paths. Freshness is relative to events
processed by RTE. Metered recalculation inherits the domain engine's snapshot rules,
including its behavior for usage arriving before an existing source snapshot.

A `CalculationRequestedEvent` on the system-events topic requests an explicit
calculation. An optional `freshness` policy specifies `track_for_seconds` and
`max_age_seconds` for a pull. Go callers can register durable push demand using
`rte.NewCalculationTarget` and `DemandService.Activate`.

## Ordering and recovery

PostgreSQL tracks dirty and completed versions per key. Workers lock one key at a time.
State, the accepted update event and completion commit together. Failed work stays
dirty and retries with backoff. Distinct source events can arrive out of order. Their
source and ID identify redeliveries during the receipt retention window. See [demand processing](demand/README.md) for lock, retry and retention details.

A strictly later `calculated_at` (calculation start) wins concurrent state writes across
replicas, so their clocks must be aligned. Each accepted write queues a
`CalculationUpdatedEvent` with its kind, key and complete persisted state. Consumers
can compare timestamps to handle duplicate or out-of-order delivery.
See [outbox delivery](../watermill/outbox/README.md).

## Running and measuring

Run `openmeter-rte --config config.yaml` with credits enabled. The worker uses
`rte.consumerGroupName`, `rte.retry`, `rte.dlq`, `rte.pollInterval` and
`rte.calculationTimeout`.
The telemetry server exposes subscription health and OpenTelemetry metrics.

| Instrument | What to compare |
| --- | --- |
| `rte.calculations`, `rte.calculation.duration` | Exact executions, outcomes and latency by calculation kind |
| `rte.demand.operations`, `rte.demand.invalidated` | Push/pull work, cached pulls, duplicate invalidations, retries and affected key count |
| `rte.upstream.events`, `rte.upstream.lag` | Source routing outcomes and time between publication and processing |
| `rte.state.reads` | Successful reads, missing state and read errors |
| `rte.rating.lookups` | Per-charge rating reuse and misses |

See the [local RTE integration and process tests](../../test/rte/README.md).
