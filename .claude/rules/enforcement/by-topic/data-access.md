# Enforcement: data-access (3 rules)

Topic file. Loaded on demand when an agent works on something in the `data-access` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Tradeoff Signals (warn)

### `dedupe-001` — The Redis deduplicator in openmeter/dedupe must be updated only after the Kafka offset commit in the sink flush. Never update Redis dedupe state before the Kafka offset commit — this breaks exactly-once semantics on consumer restart.

*source: `deep_scan`*

**Why:** The Sink Worker pattern states: 'Flush ordering is strict: ClickHouse insert then Kafka offset commit then Redis dedupe.' If Redis dedupe is updated before the Kafka offset commit and the sink crashes, the consumer restarts from the uncommitted offset but Redis marks those events as already processed, silently dropping them from ClickHouse.

**Example:**

```
// Correct three-phase order:
// 1. clickhouseStorage.BatchInsert(ctx, messages)
// 2. kafkaConsumer.CommitOffsets(ctx)
// 3. deduplicator.Set(ctx, keys)

// Wrong: updating Redis before offset commit loses events on crash
```

## Pattern Divergence (inform)

### `ingest-001` — CloudEvent ingestion must flow through openmeter/ingest.Collector (which handles deduplication and Kafka forwarding). Never write ingest events directly to ClickHouse from domain code.

*source: `deep_scan`*

**Why:** The openmeter/ingest component description states: 'Collector interface (Ingest, Close) receives single events and forwards to Kafka. DeduplicatingCollector wraps any Collector with Redis or in-memory deduplication.' Usage events must flow through the deduplication layer before reaching ClickHouse via the sink-worker. Bypassing Collector skips deduplication, causes double-counting on retries, and breaks the three-phase sink flush ordering.

**Example:**

```
// Correct: domain code calls ingest.Collector
return collector.Ingest(ctx, namespace, event)

// Wrong: inserting directly into ClickHouse from a domain service
return clickhouseClient.Exec(ctx, "INSERT INTO events ...")
```

### `meter-001` — Meter value and group-by extraction from CloudEvent JSON at ingest time must use meter.ParseEvent. Do not re-implement JSON path extraction outside this function.

*source: `deep_scan`*

**Why:** The openmeter/meter component description states: 'ParseEvent extracts value and group-by fields from CloudEvent JSON at ingest time.' ParseEvent centralises JSONPath evaluation, aggregation type dispatch, and event schema validation. Duplicating this logic elsewhere diverges from the meter definition and can produce inconsistent aggregations in ClickHouse.

**Example:**

```
// Correct: use meter.ParseEvent during ingest processing
parsed, err := meter.ParseEvent(m, event)
if err != nil { return err }
// parsed.Value, parsed.GroupByValues are safe to use

// Wrong: re-implementing JSONPath extraction inline in sink code
```

**Path glob:** `openmeter/sink/**/*.go`, `openmeter/ingest/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "jsonpath|gjson\\.Get|json\\.Path"
    ],
    "must_not_match": [
      "meter\\.ParseEvent",
      "ParseEvent"
    ]
  }
]
```

</details>
