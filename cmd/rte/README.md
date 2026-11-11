# Running the RTE worker

Build the executable from the repository root and validate the deployment configuration before
starting it.

```sh
make build-rte
./build/rte --config config.yaml --validate
./build/rte --config config.yaml
```

`--validate` checks the configuration and requires `credits.enabled`. Startup initializes the
domain services for both calculators, runs the configured migrations, and starts the Kafka worker
and telemetry server. The two source topics must have different names. The shared system-event
outbox publisher drains committed updates to Kafka. Its delivery limits and retries use the
[outbox configuration](../../openmeter/watermill/outbox/README.md).

| Setting | Effect |
| --- | --- |
| `events.systemEvents.topic`, `events.ingestEvents.topic` | Inputs for calculation requests, domain facts and stored-usage notifications |
| `rte.consumerGroupName` | Kafka group shared by RTE replicas, default `om_rte` |
| `rte.retry`, `rte.dlq` | Message-processing retries and dead lettering |
| `rte.pollInterval` | Idle or failed background polling cadence, default 1s |
| `rte.calculationTimeout` | Time budget for one demand calculation, default 30s |
| `rte.publishSourceEvents` | Set on **billing and ledger writer processes** to enable additional credit source facts. Defaults to false. |

On the telemetry listener, `/healthz/subscriptions` reports Kafka subscription status for both
topics. Use the [framework metrics](../../openmeter/rte/README.md) to distinguish a healthy
subscription from actual source routing and successful calculations. The
[local process test](../../test/rte/README.md) exercises the compiled worker with PostgreSQL,
Kafka and ClickHouse.
