# RTE tests

The ordinary suite uses PostgreSQL with service-backed billing/ledger fixtures.
The optional live test starts the production RTE executable as a separate process,
with real Kafka and ClickHouse. It checks credit/metered push calculations, cached
pulls, targeted ledger updates, late usage, rating reuse and source redelivery.
It only accepts loopback services and creates isolated PostgreSQL/ClickHouse
databases, Kafka topics, consumer group and telemetry port.

From the repository root:

```sh
docker compose up -d postgres kafka clickhouse
nix develop --impure .#ci -c make build-rte
POSTGRES_HOST=127.0.0.1 \
  RTE_E2E_BINARY="$PWD/build/rte" \
  RTE_E2E_CONFIG="$PWD/test/rte/live.yaml" \
  nix develop --impure .#ci -c go test -tags=dynamic ./test/rte -run '^TestRTELiveProcess$' -count=1 -v
```

Run the two Nix commands sequentially. The test prints its child process log path;
on failure it preserves that log and the generated configuration in a temporary
directory. Fixture setup uses the real domain services. Usage is inserted through
the ClickHouse connector, followed by the sink's ingest notification on Kafka;
this covers the RTE boundary, not the ingestion HTTP endpoint or sink process.
