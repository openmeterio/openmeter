# Migrating the ClickHouse events table to `ReplicatedMergeTree`

OpenMeter creates the events table (`om_events` by default) at startup using
the `MergeTree` storage engine. ClickHouse Cloud transparently rewrites this
to `SharedMergeTree`, so cloud users do not need to act. Self-hosted multi-node
ClickHouse clusters that require replication should switch to
`ReplicatedMergeTree`. This guide covers both new and existing deployments.

## Configuration

The relevant config block is `aggregation.clickhouse.eventsTableEngine`:

```yaml
aggregation:
  clickhouse:
    address: clickhouse:9000
    database: openmeter
    eventsTableEngine:
      # MergeTree (default) | ReplicatedMergeTree
      type: ReplicatedMergeTree
      # Required when type is ReplicatedMergeTree. Supports ClickHouse macros.
      zooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}"
      # Required when type is ReplicatedMergeTree. Typically a macro.
      replicaName: "{replica}"
      # Optional. Renders ON CLUSTER `{cluster}` on the CREATE TABLE statement.
      # The cluster name is backtick-quoted by OpenMeter, so any value
      # ClickHouse accepts in <remote_servers> works (including hyphens, dots).
      cluster: prod-cluster-1
```

Validation runs at startup. If `type` is `ReplicatedMergeTree`, both
`zooKeeperPath` and `replicaName` must be set; otherwise the service refuses
to start.

`{shard}`, `{replica}`, `{database}`, and `{table}` are ClickHouse server-side
macros — they are passed through verbatim and substituted by ClickHouse using
the per-host `<macros>` definitions in `config.xml`. See the upstream
documentation for
[Replication](https://clickhouse.com/docs/engines/table-engines/mergetree-family/replication#creating-replicated-tables).

## New deployments

Add the `eventsTableEngine` block to your config before first startup. The
service will create `om_events` with the configured engine; nothing else is
required.

## Existing deployments

`CREATE TABLE IF NOT EXISTS` is a no-op against an existing table, so simply
flipping the config does **not** convert an existing `MergeTree` table into a
`ReplicatedMergeTree` one. The conversion is offline and must be done
manually. Pause ingestion (stop the sink-worker and any direct producers)
before starting.

> **Distributed DDL is not atomic across the cluster.** Every `ON CLUSTER`
> statement is queued per-host and executed asynchronously. A node that is
> down or slow will lag, and partial states are observable until it catches
> up. Between every step below, drain the DDL queue before moving on:
>
> ```sql
> SELECT host_name, status, exception_text
> FROM system.distributed_ddl_queue
> WHERE entry > now() - INTERVAL 1 HOUR
> ORDER BY entry DESC;
> ```
>
> Wait until every replica reports `status = 'Finished'` (or investigate any
> `exception_text`) before issuing the next statement.

### Confirm the live engine before and after

Before starting, capture the current state so you can confirm the swap landed:

```sql
SELECT engine, engine_full
FROM system.tables
WHERE database = 'openmeter' AND name = 'om_events';
```

Run the same query after step 6 to verify the engine is now
`ReplicatedMergeTree` (or `Replicated...` with the full clause matching what
OpenMeter would emit).

### Migration steps

1. Stop ingestion so no rows arrive during the swap.
2. Rename the live table out of the way:
   ```sql
   RENAME TABLE openmeter.om_events TO openmeter.om_events_legacy ON CLUSTER {cluster};
   ```
   Verify the rename completed on every replica via
   `system.distributed_ddl_queue` (see the warning above) before continuing.
3. Create the replicated table with the schema OpenMeter uses (copy the
   CREATE statement OpenMeter would emit on a fresh start; you can grab it
   by running the service against an empty database, or by inspecting the
   unit tests in `openmeter/streaming/clickhouse/event_query_test.go`):
   ```sql
   CREATE TABLE openmeter.om_events ON CLUSTER {cluster} (
       namespace String,
       id String,
       type LowCardinality(String),
       subject String,
       source String,
       time DateTime,
       data String,
       ingested_at DateTime,
       stored_at DateTime,
       INDEX om_events_stored_at stored_at TYPE minmax GRANULARITY 4,
       store_row_id String
   )
   ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}')
   PARTITION BY toYYYYMM(time)
   ORDER BY (namespace, type, subject, toStartOfHour(time));
   ```
   Again, drain `system.distributed_ddl_queue` before continuing — the table
   must exist on every replica before backfill.
4. Backfill from the legacy table. For large datasets, do this one partition
   at a time so failures are recoverable:
   ```sql
   INSERT INTO openmeter.om_events
   SELECT * FROM openmeter.om_events_legacy
   WHERE toYYYYMM(time) = 202601;
   ```
   Repeat for each `toYYYYMM(time)` partition. Use
   `system.parts` on the source table to enumerate them. After each batch,
   wait for replication to settle:
   ```sql
   SELECT database, table, queue_size, inserts_in_queue, merges_in_queue
   FROM system.replicas
   WHERE database = 'openmeter' AND table = 'om_events';
   ```
   Move on once `queue_size` reaches zero on every replica.
5. Verify row counts match per partition:
   ```sql
   SELECT toYYYYMM(time) AS p, count() FROM openmeter.om_events GROUP BY p ORDER BY p;
   SELECT toYYYYMM(time) AS p, count() FROM openmeter.om_events_legacy GROUP BY p ORDER BY p;
   ```
6. Update the OpenMeter config with the new `eventsTableEngine` block and
   restart the services. The `CREATE TABLE IF NOT EXISTS` on startup is a
   no-op against the new replicated table; it is there to keep first-time
   bootstrap working. Run the `system.tables` engine check above to confirm.
7. Resume ingestion. Once you are confident in the new table, drop the legacy
   one:
   ```sql
   DROP TABLE openmeter.om_events_legacy ON CLUSTER {cluster} SYNC;
   ```

If you run a single-shard cluster you can use `EXCHANGE TABLES` instead of
`RENAME` + recreate to make the swap atomic on a single node, but the
backfill must then complete before the swap rather than after, and the
distributed-DDL caveat still applies if you `EXCHANGE TABLES ... ON CLUSTER`.

## Local test stack

A 2-node replicated ClickHouse cluster (one shard, two replicas, one Keeper)
is provided at the repository root for testing this code path locally:

```bash
make up-replicated
# Native ports: 127.0.0.1:39000 (clickhouse-01), 127.0.0.1:39001 (clickhouse-02)
# HTTP ports:   127.0.0.1:38123 (clickhouse-01), 127.0.0.1:38124 (clickhouse-02)
# Keeper:       127.0.0.1:39181

# Run the integration tests (gated by TEST_CLICKHOUSE_REPLICATED):
TEST_CLICKHOUSE_DSN=clickhouse://default:default@127.0.0.1:39000/openmeter \
TEST_CLICKHOUSE_REPLICATED=1 \
  go test -tags=dynamic -run='TestEventsTableEngine$' \
  ./openmeter/streaming/clickhouse/...

make down-replicated
```

The cluster name is `openmeter_cluster` and the macros (`{shard}`,
`{replica}`, `{cluster}`) are pre-defined on each node, so the OpenMeter
config that targets it looks like:

```yaml
aggregation:
  clickhouse:
    address: 127.0.0.1:39000
    eventsTableEngine:
      type: ReplicatedMergeTree
      zooKeeperPath: "/clickhouse/tables/{shard}/{database}/{table}"
      replicaName: "{replica}"
      cluster: openmeter_cluster
```

## Notes

- Backwards compatible: the new config field is optional. Existing
  deployments that do not set `eventsTableEngine` continue to use
  `ENGINE = MergeTree`, exactly as before.
- ClickHouse Cloud users should leave the engine on the default `MergeTree`;
  Cloud rewrites it to `SharedMergeTree` server-side.
- Materialized views and other tables are not currently created by OpenMeter
  itself, so this migration only concerns `om_events`.
