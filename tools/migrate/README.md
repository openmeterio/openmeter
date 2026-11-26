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

Use three separately deployed releases for the Notification storage transition:

1. Deploy the [dual-reader changes (#5638)](https://github.com/openmeterio/openmeter/pull/5638)
   to every API and worker. These still write Rule feature strings and typed Event envelopes.
2. Deploy the writer release (`fix/notification-canonical-writers`) everywhere. It writes
   Feature ID objects and bare, snake_case Event snapshots. Wait for older processes
   and their in-flight writes to drain.
3. Deploy the data migrations in #5639 after running their preflights. Each data
   backfill preserves resource identity and lifecycle fields.

The writer PR is stacked on #5638; #5639 is stacked on the writer PR. Merging
these together into one release would bypass the reader rollout requirement.
Rollback after the writer release must stop at the dual-reader release. Its old-format
writes would require another backfill before V4 takes over.

After Rule config normalization, run the [read-only preflight](notification_rule_feature_ids_preflight.sql).
For a live Rule, a key must identify the feature active at its last update.
Soft deletion also advances that timestamp, so for deleted Rules the preflight
considers every feature with that key active between Rule creation and deletion.
Multiple candidates block the backfill. Use the candidate IDs and historical
evidence to repair the reference, then rerun the preflight.

The backfill stores each resolved Feature as `{"id":"…"}`. Existing object references
match by ID only, including when another feature uses that ID as its key. Mixed
lists retain the first occurrence of each ID. An unrestricted JSON null becomes
an omitted field, which the V4 contract accepts.

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

## Notification Rule threshold types

After the V1 writer rollout, the threshold migration converts `NUMBER` to
`usage_value` and `PERCENT` to `usage_percentage` in balance-threshold Rules,
including deleted Rules. It preserves threshold order, Rule revisions, and
timestamps. Historical Event payloads keep their original values. Downgrading
keeps the canonical types, which the previous release already accepts.

## Notification Event payload snapshots

Run [the Event payload preflight](notification_event_payload_snapshots_preflight.sql)
after the writer rollout. `legacy_envelope` rows need conversion; `invalid` rows
block the migration and require repair. Unexpected envelope fields also block
conversion so they cannot be silently discarded. Unknown Event types are skipped.

The backfill extracts each stored type-specific snapshot and converts model field
names to snake_case. It preserves exact JSON numbers, array order, and keys inside
metadata, annotations and meter filters. Event IDs, timestamps, Rule references,
snapshots, annotations, provider data and delivery records remain unchanged.
It does not fetch live billing or entitlement state. Canonical rows are not rewritten;
rerunning is safe. Downgrading preserves the payloads, which the prerequisite
reader release already supports.

The conversion runs transactionally and can hold row locks until completion.
Size the historical Event set before scheduling the backfill. This does not alter
Svix messages already accepted by the provider or enable V4 routing.
