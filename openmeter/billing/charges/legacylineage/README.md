# Legacy credit-realization lineage — deprecated

Compatibility only: new collections use [ledger origins](../../../ledger/provenance-design.md)
and derive amounts from ledger entries. Do not add new lineage consumers or
create roots and segments for origin-tracked collections.

Pre-cutover histories still need both reads and writes here: purchases backfill
their advances, recognition updates their segments, and corrections unwind them.
Mixed histories must retain one collection-time FIFO order and persist the exact
legacy segments and amounts selected by ledger posting.

The stored table names, enum values, and billing annotations remain unchanged.
Deprecation does not make these records disposable; remove this compatibility
path only when legacy histories can no longer require lifecycle operations.
