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
index. The index build blocks delivery writes while it scans the table, so schedule
the upgrade accordingly. If duplicates remain, PostgreSQL rejects the index and
the transaction rolls back, preserving the original index and delivery history.
