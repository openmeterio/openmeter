{{ template "chart.baseHead" . }}


## Getting started

OpenMeter needs meters to be defined in configuration to process events:

```yaml
config:
  meters:
    - slug: m1
      description: API calls
      eventType: api-calls
      valueProperty: $.duration_ms
      aggregation: SUM
      groupBy:
        method: $.method
        path: $.path
```

See [values.example.yaml](values.example.yaml) for more details.

> Values defined in `config` will get overwritten by the values calculated from chart values!

## Running OpenMeter in production

This Helm chart comes with a default setups for Kafka, ClickHouse, Postgres, Redis and Svix.

**It is highly recommended to use your own dependencies in production.**

You can disable installing the above dependencies to bring your own:

```yaml
svix:
  enabled: false

redis:
  enabled: false

postgres:
  enabled: false

kafka:
  enabled: false

clickhouse:
  enabled: false
```

In this case, you need to provide the connection details in `config`:

```yaml
config:
    ingest:
      kafka:
        broker: KAFKA_ADDRESS

    aggregation:
      clickhouse:
        address: CLICKHOUSE_ADDRESS
        username: default
        password: ""
        database: default

    postgres:
      url: postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
      autoMigrate: migration

    svix:
      apiKey: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdml4LXNlcnZlciIsInN1YiI6Im9yZ18yM3JiOFlkR3FNVDBxSXpwZ0d3ZFhmSGlyTXUiLCJleHAiOjE4OTM0NTYwMDAsIm5iZiI6MTcwNDA2NzIwMCwiaWF0IjoxNzIzNTUzMTQ0fQ.JVOFgHymisTD-Zw_p03qD4iUXXXw-VwABda2Q3f1wfs
      serverURL: http://127.0.0.1:8071/
      debug: true
```

## Per-component scheduling and resources

Each OpenMeter component reads its own `resources`, `nodeSelector`, `tolerations`, and `affinity` values. A component value replaces the equivalent chart-wide value. If you do not set a component value, the chart-wide value applies.

The components are `api`, `balanceWorker`, `notificationService`, `sinkWorker`, `billingWorker`, `svix`, `jobs.subscriptionSync`, `jobs.billingCollectInvoices`, and `jobs.billingAdvanceInvoices`.

```yaml
# Chart-wide default for every component.
nodeSelector:
  workload: openmeter
resources:
  requests:
    cpu: 100m
    memory: 256Mi

# The sink worker needs more memory and a dedicated node pool.
sinkWorker:
  nodeSelector:
    workload: openmeter-sink
  tolerations:
    - key: openmeter-sink
      operator: Exists
      effect: NoSchedule
  resources:
    limits:
      cpu: "2"
      memory: 4Gi
    requests:
      cpu: "1"
      memory: 2Gi

# The cron jobs run on spot nodes.
jobs:
  billingCollectInvoices:
    nodeSelector:
      workload: openmeter-batch
```

{{ template "chart.valuesSection" . }}
