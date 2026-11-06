# RTE

RTE stores explicitly requested current credit and metered entitlement balance
calculations. Credit balance calculation calls
`customerbalance.Service.GetBalance` without `AsOf`, preserving the existing
booked, spendable, live, and pending semantics. It does not change existing
customer balance endpoints or the entitlement worker.

RTE owns the typed calculate/read lifecycle, timestamped state, and persistence.
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

The process currently reuses the calculator’s existing service wiring, including
Kafka, ClickHouse, and the shared system-event outbox publisher. It does not
subscribe to events.

Run the separate process with the usual OpenMeter configuration and credits enabled:

```sh
go run -tags=dynamic ./cmd/rte --config config.yaml
```

The Go service exposes explicit calculate and read operations. The process shell
runs through the shared application runner; a request transport is not selected
yet. No balance routes or event consumers are registered.
