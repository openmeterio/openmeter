## Schema source

Atlas runs `cmd/entschema`, which uses the reusable `entschema.GenerateSQL`
exporter with a schema path, PostgreSQL version, and supplemental DDL.
[`supplemental_schema.sql`](supplemental_schema.sql) defines deferred foreign keys
and indexes on implicit join tables that Ent cannot express.

## View SQL Helper

Generate SQL definitions for `ent.View` schemas:

```bash
make generate-view-sql
```

`viewgen.GenerateSQL` returns PostgreSQL `CREATE VIEW` statements from EntSQL
annotations. The command adds the regeneration header and writes
`tools/migrate/views.sql`.

## Notification Rule feature ID backfill

Upgrade through the release containing [the V1 Rule writer](https://github.com/openmeterio/openmeter/pull/5638)
and drain older writers **before deploying the release containing this migration**.
With automatic migrations enabled, the first new server can run the backfill
while old replicas are still serving writes.

After Rule config normalization, run the [read-only preflight](notification_rule_feature_ids_preflight.sql).
For a live Rule, a key must identify the feature active at its last update.
Soft deletion also advances that timestamp, so for deleted Rules the preflight
considers every feature with that key active between Rule creation and deletion.
Multiple candidates block the backfill. Use the candidate IDs and historical
evidence to repair the reference, then rerun the preflight.

Missing references are dropped. If every reference is missing, an enabled Rule
stays enabled and matches every feature. The preflight flags that case separately.
Downgrading keeps the resolved IDs and does not restore dropped references.

## Notification delivery destination uniqueness

Run the [duplicate preflight](notification_delivery_duplicate_preflight.sql) before
upgrading. Each result lists the delivery IDs for one namespace, Event, and
Channel. Inspect their attempts and resolve duplicates before applying the unique
index. The first migration builds `notification_delivery_destination_unique`
concurrently. Once it is valid, a second migration drops the old lookup index
concurrently. Each migration executes a single statement, so deliveries can
continue throughout the swap. Downgrading restores the lookup index before
removing uniqueness.

If a duplicate appears after preflight, the build can fail with
`schema_om` dirty at `20261125171748`. The old index still serves lookups.
Inspect the replacement before retrying:

```sql
SELECT indisvalid, indisready, pg_get_indexdef(indexrelid)
FROM pg_index
WHERE indexrelid = to_regclass('notification_delivery_destination_unique');
```

For an invalid replacement, run
`DROP INDEX CONCURRENTLY notification_delivery_destination_unique` outside a
transaction. Rerun the duplicate preflight and repair any conflicts. After
verifying the old index is valid and the replacement is absent, use
golang-migrate with `x-migrations-table=schema_om` to force the preceding
version (`20261125160000`) and rerun the migrations. If the replacement is
already valid and matches the intended unique definition, the build completed.
Record the build version as clean and resume the migration that removes the old
index.

## Notification Channel Event type filters

Deploy #5638 on all V1 writers and drain earlier writers and in-flight transactions.
After Rule type normalization, run the [Channel filter preflight](notification_channel_event_type_filters_preflight.sql).
It shows the stored and expected filters, flags rows the backfill will update, and
lists enabled Channels without a live Rule assignment. Disabled Rules count as assignments.

The backfill locks live Channels in ID order and may delay assignment or
Channel changes until commit. It writes only changed filters and preserves enabled
state, timestamps, soft-deleted Channels and Svix routing. Downgrading keeps the
projection. At V4 routing cutover, fence assignment changes, rerun the preflight,
and disable the legacy Channels flagged as catch-all before switching routing.
