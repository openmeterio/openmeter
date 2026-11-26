# Running the RTE worker

RTE is disabled by default. A disabled worker exits before opening connections or running
migrations. To run it, enable both `rte.enabled` and `credits.enabled` in the configuration.
Use distinct Kafka topics for system events and stored-usage notifications.

From the repository root, build the worker, validate the configuration, then start it.

```sh
make build-rte
./build/rte --config config.yaml --validate
./build/rte --config config.yaml
```

Startup runs the configured migrations and starts message consumption, demand processing and
the telemetry server. An unexpected worker or listener failure causes a nonzero exit.
Individual message and calculation failures follow their respective retry policies.

| Setting | What to tune |
| --- | --- |
| `rte.consumerGroupName` | Kafka group shared by replicas. Defaults to `om_rte`. |
| `rte.retry`, `rte.dlq` | Retries and dead lettering for message handlers. With the DLQ disabled, failed messages remain eligible for broker redelivery after retries. |
| `rte.pollInterval` | Delay after an idle background slot finds no work or fails to claim it. Defaults to 1s. |
| `rte.calculationConcurrency` | Background slots per process. Defaults to 4. Message handlers can calculate concurrently with those slots. |
| `rte.incrementalCalculations` | Allow demand refreshes to reuse [verified calculation checkpoints](../../openmeter/rte/service/checkpoints.md). Defaults to false. |
| `rte.calculationTimeout` | Time budget for an individual demand calculation. Defaults to 30s. |
| `rte.shadowSamplePercent` | Percentage of balance keys sampled by producers during the trial. Accepts 1–100 and defaults to 10. |

The Helm chart deploys the worker when `config.rte.enabled` is true. On its telemetry listener,
`/healthz/subscriptions` reports whether it has subscribed to both topics. Use the routing and
calculation metrics to see whether keys are being refreshed. RTE replicas deliver and retry updates from their own outbox queue. [Metrics](../../openmeter/rte/metrics/README.md#instruments) ·
[Outbox delivery](../../openmeter/watermill/outbox/README.md)
