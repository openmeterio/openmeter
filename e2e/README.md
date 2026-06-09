# E2E Tests & Benchmarks

End-to-end tests and benchmarks that run against a **live OpenMeter stack** over HTTP.
Both skip unless `OPENMETER_ADDRESS` is set.

## Run the local stack

```sh
make env-local-up      # build + start openmeter + infra (docker compose)
make env-local-down    # tear down
```

Server listens on `http://localhost:38888`. `config.yaml` is bind-mounted, so config
changes need only `make env-local-up` (force-recreate), not a full down/up. Logs land in
`e2e/logs/`.

## Benchmarks

`BenchmarkGovernanceQuery` measures `POST /api/v3/openmeter/governance/query` latency
across a customers × features grid. Seeds boolean entitlements (no usage events).

```sh
make bench-governance          # 1x1 baseline + 10/50/100 diagonal
make bench-governance-matrix   # full 3x3 customers x features matrix
```

Overridable vars (defaults shown):

| Var                 | Default                  | Purpose                                      |
|---------------------|--------------------------|----------------------------------------------|
| `OPENMETER_ADDRESS` | `http://localhost:38888` | target server                                |
| `BENCHTIME`         | `20x`                    | iterations per sub-benchmark                 |
| `COUNT`             | `1`                      | repeat count (use >1 for benchstat variance) |

For variance / before-after comparison:

```sh
make bench-governance COUNT=10 | tee baseline.txt
benchstat baseline.txt                 # mean ± %CV
benchstat baseline.txt after.txt       # delta + p-value
```

> `ns/op` = mean of sequential request latencies (no concurrency, no tail percentiles).
> Boolean entitlements skip the metered/ClickHouse balance path — a relative algorithmic
> baseline, not a production-latency oracle.

## Traces (optional)

OpenMeter runs in a container, so OTLP must target the **host**, not container loopback.
Point the OTLP exporters in `e2e/config.yaml` at `host.docker.internal:4317` and run a
collector on the host (e.g. `grafana/otel-lgtm`). Then query per-size latency percentiles
in Grafana (Tempo, TraceQL metrics):

```
{ name = "governance.QueryAccess" && span.customer_key_count = 100 && span.feature_key_count = 100 }
  | quantile_over_time(duration, 0.5, 0.9, 0.95, 0.99)
```

High-fan-out requests emit thousands of SQL spans; bump Tempo's `max_bytes_per_trace`
(default 5MB) or traces get dropped at ingest.

## Standard e2e tests

```sh
make test-local        # full down → up → go test ./... → down
```
