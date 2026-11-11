# RTE tests

The ordinary suite uses PostgreSQL with service-backed billing/ledger fixtures.
The optional live test starts the production RTE executable as a separate process,
with real Kafka and ClickHouse. It checks credit/metered push calculations, cached
pulls, targeted ledger updates, late usage, rating reuse and source redelivery.
It only accepts loopback services and creates isolated PostgreSQL/ClickHouse
databases, Kafka topics, consumer group and telemetry port.

## Running locally

Run the service-backed suite with PostgreSQL available. The event pipeline tests
exercise the real router and transactional outbox with in-memory message transport.

```sh
POSTGRES_HOST=127.0.0.1 nix develop --impure .#ci -c go test -tags=dynamic ./test/rte -run '^TestRTEIntegration$' -count=1
```

To run the separate-process scenario, execute these commands from the repository
root:

```sh
docker compose up -d postgres kafka clickhouse
nix develop --impure .#ci -c make build-rte
POSTGRES_HOST=127.0.0.1 \
  RTE_E2E_BINARY="$PWD/build/rte" \
  RTE_E2E_CONFIG="$PWD/test/rte/live.yaml" \
  nix develop --impure .#ci -c go test -tags=dynamic ./test/rte -run '^TestRTELiveProcess$' -count=1 -v
```

Run the two Nix commands sequentially. The test prints its child process log path.
On failure it preserves that log and the generated configuration in a temporary
directory. Fixture setup uses the real domain services. Usage is inserted through
the ClickHouse connector, followed by the sink's ingest notification on Kafka.
The test begins at the stored-usage notification boundary.

## What the process test observes

| Step | Observable result |
| --- | --- |
| Activate a credit key and a metered entitlement key | The worker persists each initial balance. |
| Request a fresh credit balance | The cached-pull metric increments and the stored timestamp stays the same. |
| Fund the customer with another 25 credits | Credit live balance reaches 125, rating records a reuse hit, and the metered timestamp stays the same. |
| Store 25 units of late usage and deliver the ingest notification | Credit stays booked at 125 and live falls to 100. Metered usage reaches 25 and the remaining entitlement balance falls to 75. |
| Redeliver the same CloudEvent | The duplicate metric increments. The calculation count and both stored states stay the same. |

The duplicate assertion waits for acknowledgement through the worker’s metric
before checking the counts. This distinguishes successfully suppressed work
from a message the worker has yet to process. For rollback, failed enqueue and
equal or older attempts, see the [transaction tests](event_transaction_test.go).
