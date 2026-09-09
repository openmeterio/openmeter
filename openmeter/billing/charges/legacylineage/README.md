# Legacy credit-realization lineage — deprecated

Compatibility only: new collections use [ledger origins](../../../ledger/README.md#transaction-invariants)
and derive amounts from ledger entries. Do not add new lineage consumers or
create roots and segments for origin-tracked collections.

Pre-cutover histories still need both reads and writes here: purchases backfill
their advances, recognition updates their segments, and corrections unwind them.
Mixed histories must retain one collection-time FIFO order and persist the exact
legacy segments and amounts selected by ledger posting.

Correction uses the collector's shared source-order planner. Its reader maps
legacy segments to original sources/backing groups; recognition time never
changes source priority. The writer requires the exact segment selections carried
on the correction realization and checks them under lineage locks. Missing or
stale selections fail and roll back the enclosing ledger/realization transaction.
Existing correction rows need no backfill: they are already persisted and are
not submitted again to this writer.

The stored table names, enum values, and billing annotations remain unchanged.
Deprecation does not make these records disposable; remove this compatibility
path only when legacy histories can no longer require lifecycle operations.

Apply the additive origin migration before switching writers; its index creation
blocks ledger writes while it runs. Stop old writers before origin-bearing
posting begins. Subsequent application rollbacks require a version that reads
and preserves v3 entry identities; retain the origin column and legacy tables.
