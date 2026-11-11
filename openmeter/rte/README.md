# RTE

RTE stores explicitly requested current credit and metered entitlement balance
calculations. Credit balance calculation calls
`customerbalance.Service.GetBalance` without `AsOf`, preserving the existing
booked, spendable, live, and pending semantics. It does not change existing
customer balance endpoints or the entitlement worker.

RTE owns the typed calculate/read lifecycle, timestamped state, persistence, and
calculation events.
Each calculation package supplies its key, result, and `Calculator` implementation.
The service constructor pairs each registered calculator with its typed store.
Callers select `CreditBalances()` or `MeteredEntitlements()` and use its `Calculate`
and `Get` operations.

Credit balance state is scoped by namespace, customer ID, currency identity
(including custom currency ID), and feature scope: all credits, unrestricted credits, or credits
eligible for one feature. Reads only return the last successful stored result.
Before any successful calculation, the result is “not calculated”, never zero.

Metered entitlement balances follow the same explicit calculate/store/read flow,
scoped by namespace and entitlement ID. They reuse `GetEntitlementBalance`,
including its minute rounding and usage-period semantics, and preserve the full
balance result. RTE's stored result is separate from the credit engine's
intermediate calculation snapshots; it is not an access decision.

`calculated_at` records calculation start. A successful result is stored with an
atomic conditional UPSERT: only a strictly later timestamp replaces existing
state. Equal or older timestamps retain and return the stored result. Failed
calculations do not write state. This ordering relies on replica clocks being
aligned; it is not a source revision or a transactional snapshot guarantee.

`CalculationRequestedEvent` selects a calculation kind and its typed key. The RTE
worker consumes these requests from the shared system-events topic in its own
consumer group (`rte.consumerGroupName`), using the standard retry and DLQ pipeline
(`rte.retry`, `rte.dlq`). Other event types do not trigger calculations.

Every accepted state write enqueues a `CalculationUpdatedEvent` in the same
transaction. It carries the kind, key, and complete persisted state. Failed,
equal-timestamp, and older attempts emit no update. Updates use the existing
system-event outbox; a successful calculation means the update is queued, not
necessarily delivered.

The pipeline can redeliver events. A repeated request can run a fresh calculation
and produce a newer update. Updates may repeat or arrive out of order; consumers can
compare `calculated_at` for the same kind and key. See the
[outbox delivery and retry semantics](../watermill/outbox/README.md). No
source-event routing or invalidation is included.

Run the separate process with the usual OpenMeter configuration and credits enabled:

```sh
go run -tags=dynamic ./cmd/rte --config config.yaml
```

The Go service exposes explicit calculate and read operations. The process uses
the shared application runner and calculator wiring. Reads remain Go-only; no
HTTP endpoints are registered.
