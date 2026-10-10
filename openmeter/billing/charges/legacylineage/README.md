# Legacy credit-realization lineage — deprecated

Compatibility for legacy lineage histories. Origin-tracked collections derive
amounts from [ledger entries](../../../ledger/README.md#collection-provenance).
Do not add lineage consumers or create roots and segments for origin-tracked
collections.

Legacy lineage histories still need both reads and writes here: purchases backfill
their advances, recognition updates their segments, and corrections unwind them.
Mixed histories must retain one collection-time FIFO order and persist the exact
legacy segments and amounts selected by ledger posting.

Correction uses the [shared source-order planner](../../../ledger/collector/correction/README.md). Its reader maps
legacy segments to original sources/backing groups; recognition time never
changes source priority. The writer requires exact segment selections in the
correction realization's `ledger.correction.legacy_segments` annotation and checks
them under lineage locks. Missing or stale selections fail and roll back the
enclosing ledger/realization transaction.
Existing correction rows need no backfill: they are already persisted and are
not submitted again to this writer.

The stored table names, enum values, and existing annotation keys are preserved.
Deprecation does not make these records disposable; remove this compatibility
path only when legacy histories can no longer require lifecycle operations.

Deployment must preserve both ledger identities and legacy state. See the
[collection-provenance migration guide](../../../../docs/migration-guides/2026-09-17-ledger-collection-provenance.md)
for schema ordering, writer compatibility, and rollback requirements.
