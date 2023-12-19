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

## Running OpenMeter in production

This Helm chart comes with a default Kafka and ClickHouse setup (via their respective operators).

**It is highly recommended to use your own Kafka and ClickHouse clusters in production.**

You can disable installing Kafka/Clickhouse and their operators to bring your own:

```yaml
kafka:
  enabled: false
  operator:
    install: false

clickhouse:
  enabled: false
  operator:
    install: false
```

In this case, you need to provide the Kafka and ClickHouse connection details in `config`:

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
```

{{ template "chart.valuesSection" . }}
