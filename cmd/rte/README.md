# Running the RTE worker

Build the executable from the repository root and validate the deployment configuration before
starting it.

```sh
make build-rte
./build/rte --config config.yaml --validate
./build/rte --config config.yaml
```

RTE defaults to disabled. The command then exits before opening connections or running migrations.
With `rte.enabled: true`, `--validate` checks the configuration and requires `credits.enabled`. Startup initializes the
domain services for both calculators, runs the configured migrations, and starts the Kafka worker
and telemetry server. The two source topics must have different names. The worker drains its own calculation updates from the shared outbox. Other system events
continue through their usual publishers. Delivery limits and retries use the
[outbox configuration](../../openmeter/watermill/outbox/README.md).

| Setting | Effect |
| --- | --- |
| `events.systemEvents.topic`, `events.ingestEvents.topic` | Inputs for calculation requests, domain facts and stored-usage notifications |
| `rte.consumerGroupName` | Kafka group shared by RTE replicas, default `om_rte` |
| `rte.retry`, `rte.dlq` | Message-processing retries and dead lettering |
| `rte.pollInterval` | Interval between polls when no key is ready or a claim fails, default 1s |
| `rte.calculationTimeout` | Time budget for one demand calculation, default 30s |
| `rte.enabled` | Starts the worker and enables shadow observations and source facts on participating processes. The Helm chart uses `config.rte.enabled` to deploy the worker. |
| `rte.shadowSamplePercent` | Percentage of balance keys observed on the API and balance workers (1–100, default 10). |

On the telemetry listener, `/healthz/subscriptions` reports Kafka subscription status for both
topics. Use the [framework metrics](../../openmeter/rte/README.md) to distinguish a healthy
subscription from actual source routing and successful calculations. The
[local process test](../../test/rte/README.md) exercises the compiled worker with PostgreSQL,
Kafka and ClickHouse.
