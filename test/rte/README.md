# RTE integration tests

Run these commands from the repository root with Docker Compose and the repository's Go
toolchain. To enter the development shell, run `nix develop --impure .#ci` once.

## PostgreSQL integration suite

```sh
docker compose up -d --wait postgres
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/rte -count=1
```

This exercises persisted balances, demand tracking and transactional update delivery. The
event pipeline uses the production router and outbox with in-memory message transport.
The separate-process test skips unless both environment variables below are set.

### Credit checkpoint timings

With local PostgreSQL running, compare full and incremental `EnsureFresh + Get` on customers
with one or 32 distinct-priority grants. One workload keeps the journal unchanged and the other
funds one additional credit before every refresh. The test checks all four amounts, reports local latency
percentiles and excludes asynchronous outbox delivery from the timer.

```sh
POSTGRES_HOST=127.0.0.1 RTE_BENCHMARK_CREDITS=1 go test -tags=dynamic ./test/rte -run '^TestRTEIntegration$/TestCreditCheckpointPerformance$' -count=1 -v
```

## Worker process with Kafka and ClickHouse

```sh
docker compose up -d --wait postgres kafka clickhouse
make build-rte
POSTGRES_HOST=127.0.0.1 \
  RTE_E2E_BINARY="$PWD/build/rte" \
  RTE_E2E_CONFIG="$PWD/test/rte/live.yaml" \
  go test -tags=dynamic ./test/rte -run '^TestRTELiveProcess$' -count=1 -v
```

The test starts the compiled worker, creates isolated databases and Kafka topics, then drives
it through the RTE client. Its inputs must be local. On failure, the test keeps the worker log
and generated configuration in the temporary directory printed in the test output.

The scenario starts with 100 credits and a 100-unit entitlement. Repeated product reads exercise
a fresh calculation and a cache hit. Adding 25 credits changes the credit balance alone. Then
25 units of late usage, priced at one credit per unit, bring live credit to 100 and entitlement
balance to 75. The test inserts usage through the ClickHouse connector and publishes the sink's
stored-usage notification. Redelivering the notification refreshes both balances from the same stored usage. [Process test](live_process_test.go)

For the opt-in Compose test that creates grants through the product API and reads the worker's
result through the client, see [RTE client e2e](../../e2e/README.md#rte-client).
