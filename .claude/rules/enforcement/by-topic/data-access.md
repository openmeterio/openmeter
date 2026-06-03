# Enforcement: data-access (8 rules)

Topic file. Loaded on demand when an agent works on something in the `data-access` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pitfalls (block)

### `pf-009-trgm-gin-index` — When exposing a v3 contains/ocontains filter on a column, add a pg_trgm GIN index for that column via a custom SQL migration before shipping the list endpoint.

*source: `deep_scan`*

**Why:** Pitfall pf_0009: v3 AIP list endpoints expose case-insensitive contains/ocontains filters that compile to leading-wildcard ILIKE '%value%' via pkg/filter (filter.go:241 maps $contains to sql.FieldContainsFold). The Ent schemas back filtered columns with plain btree indexes (customer.go:64-65 declares only index.Fields("name")), so every such filtered list request degrades to a full sequential scan plus a COUNT(*) scan from query.Paginate. customer.go:42-55 carries an explicit TODO(DoS hardening) about this.

**Example:**

```
-- custom migration before exposing the v3 list filter
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX customers_name_trgm_idx ON customers USING gin (lower(name) gin_trgm_ops) WHERE deleted_at IS NULL;
```

**Path glob:** `openmeter/ent/schema/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "FieldContainsFold|contains|ocontains"
    ],
    "must_not_match": [
      "gin_trgm_ops",
      "pg_trgm"
    ]
  }
]
```

</details>

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

### `data-rawevent-clickhouse-column-sync` — When adding a field to streaming.RawEvent, update the createEventsTable DDL builder and the INSERT column list in event_query.go in the same change; column order must match the ClickHouse table exactly, and reads must go through streaming.Connector query-structs with toSQL(), never inline SQL.

*source: `deep_scan`*

**Why:** The RawEvent data model lifecycle states: 'Add a ch: tagged field to streaming.RawEvent in openmeter/streaming/connector.go, then add the column to the DDL in createEventsTable.toSQL() and to the INSERT column list in event_query.go (column order must match the table exactly).' ClickHouse schema is created by the connector at startup (createTable), not by Atlas/golang-migrate; there is a single shared table across all namespaces, so a mismatched column list silently corrupts inserts.

**Example:**

```
rows, err := connector.QueryMeter(ctx, namespace, meter, params)
```

**Path glob:** `openmeter/streaming/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "sb\\.Select\\(|fmt\\.Sprintf\\(\"SELECT|fmt\\.Sprintf\\(\"INSERT"
    ],
    "must_not_match": [
      "toSQL\\(\\)"
    ]
  }
]
```

</details>

### `data-dedupe-keyhash-rotation` — Never silently change dedupe.GetKeyHash (xxh3-128 + base64url) or the dedup key encoding; introduce a new DedupeMode and use keyhash-migration mode that checks both old rawkey and new hashed key during rollout.

*source: `deep_scan`*

**Why:** The dedupe.Item data model lifecycle states: 'Dedup keys are not schema-migrated. Changing the key encoding requires adding a new DedupeMode and updating the mode-switch in every method (IsUnique/CheckUnique/Set/CheckUniqueBatch) plus a key rotation plan' and 'Never change GetKeyHash (xxh3-128 + base64url) silently — use keyhash-migration mode which checks both old rawkey and new hashed key during rollout.' A silent change makes every previously-deduplicated event look new (or vice versa), breaking exactly-once semantics across the rollout.

**Example:**

```
switch d.Mode {
case DedupeModeRawKey: keys = append(keys, item.RawKey())
case DedupeModeKeyHashMigration: keys = append(keys, item.RawKey(), item.HashedKey())
}
```

**Path glob:** `openmeter/dedupe/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "GetKeyHash"
    ]
  }
]
```

</details>

### `data-read-through-service-not-entdb` — Read every Ent-backed entity through its domain Service composite interface backed by the adapter wrapped in entutils.TransactingRepo; never query openmeter/ent/db directly from a service or any caller outside the adapter.

*source: `deep_scan`*

**Why:** Every data model's how_to_read lifecycle says the same thing, e.g. BillingInvoice: 'Always through billing.Service (composite interface) backed by the Ent adapter wrapped in entutils.TransactingRepo; never query openmeter/ent/db directly from a service.' Direct ent/db access from a service bypasses the adapter interface (breaking unit-test mockability) and the ctx-bound transaction rebinding done by TransactingRepo.

**Example:**

```
return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) {
    return toDomain(tx.db.Entity.Get(ctx, id))
})
```

**Path glob:** `openmeter/**/service/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "entdb\\.|ent/db"
    ]
  }
]
```

</details>

### `persist-clickhouse-no-inline-sql` — All ClickHouse access must go through streaming.Connector with query logic in clickhouse/ query-structs that expose toSQL(); never inline SQL strings in connector method bodies, and remember the single events table is shared across all namespaces with namespace as the leading ORDER BY column.

*source: `deep_scan`*

**Why:** The clickhouse_events persistence store states it is a 'Single shared append-only MergeTree events table across all namespaces (namespace is the leading ORDER BY column); table DDL is created by the connector at startup, not by Atlas.' The RawEvent lifecycle adds: 'query logic lives in clickhouse/ query-structs with toSQL(), never inline SQL in connector method bodies.' Inline SQL bypasses namespace scoping and the centralized query builders, risking cross-tenant leakage.

**Example:**

```
rows, err := connector.ListEventsV2(ctx, namespace, params) // query built via clickhouse/event_query_v2.go toSQL()
```

**Path glob:** `openmeter/streaming/clickhouse/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\\.Query\\(ctx, \"|\\.Exec\\(ctx, \"|fmt\\.Sprintf\\(\"SELECT"
    ],
    "must_not_match": [
      "toSQL\\(\\)"
    ]
  }
]
```

</details>
