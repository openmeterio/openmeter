# Ledger collection provenance

Collection provenance replaces mutable lineage amounts for origin-tracked
collections. Legacy histories retain their lineage state; both formats can
coexist for a customer or charge. The ledger remains the source of accounting
amounts. See [collection provenance](../../openmeter/ledger/README.md#collection-provenance)
and [legacy compatibility](../../openmeter/billing/charges/legacylineage/README.md).

## Schema and historical data

The [schema migration](../../tools/migrate/migrations/20260921103155_ledger_entry_collection_origin.up.sql)
adds nullable `ledger_entries.collection_origin_id` and an index on namespace
and origin. It does not rewrite existing entries, identity keys, realizations,
or lineage records. No historical origin backfill is required.

The index is created without `CONCURRENTLY`; migration execution blocks writes
to `ledger_entries`. Allow for that when scheduling the migration.

Origin-bearing entries use v3 entry identity keys. Readers also accept v1/v2
identities for legacy and unrelated entries. An absent origin is not enough to
identify a lineage-managed history: ordinary purchase and payment entries also
have no collection origin.

## Writer transition

1. Apply the additive schema migration before starting origin-aware writers.
2. Drain writers that do not support collection origins, including background
   charge processing, before allowing origin-bearing postings. Do not overlap
   incompatible writer versions.
3. Run collection, advance backfill, recognition, and correction with the
   origin-aware implementation and its legacy compatibility services together.

The collector marks billing allocations with `ledger.origin_tracked = true`,
which excludes them from legacy root creation. Corrections choose their path
from the original ledger entries. Backfill and recognition handle both formats;
legacy updates persist the exact amounts selected for ledger posting in the
same transaction. Existing persisted corrections need no annotation backfill.

## Rollback

After an origin-bearing entry has been committed, an application rollback must
still understand v3 identities, preserve collection origins on downstream
postings, and support both history formats. Retaining the schema alone does not
make a lineage-only application version compatible.

Do not run the down migration while origin-bearing history exists: it drops the
origin column and index. Keep legacy tables and lifecycle services for as long
as legacy collections can still be backfilled, recognized, or corrected.
