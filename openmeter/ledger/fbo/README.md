# FBO consumption

This package applies customer credit to usage or outstanding receivables. It
selects credit sources, releases their scheduled breakage as they are consumed,
and coordinates [advances](../advance/README.md). Billing decides the amount and
settlement policy; the collector commits the transactions and creates allocations.

`Consume` returns resolved transaction inputs for one collection request. The
caller supplies the group ID and keeps source selection, breakage bookkeeping,
and group posting in one database transaction. Breakage releases reference
consumption entry IDs assigned before posting. Any failure requires rollback.

Sources are selected by priority, restricted before unrestricted credit, earliest
expiry, then stable source cursor. `SourceBalanceAsOf` is the source-selection
cutoff; `BookedAt` is the effective time of consumption. Receivable coverage never
creates advance.

`GetSpendableBalanceBuckets` reports positive capacity available at `AsOf` without
reducing any later recorded balance below zero. It takes the minimum per source
before combining buckets. Future breakage is excluded because consumption
releases it; breakage at or before `AsOf` remains part of the opening balance.
