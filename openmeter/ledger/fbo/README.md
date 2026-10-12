# FBO consumption

This package applies customer credit to usage or outstanding receivables. It
selects credit sources, releases their scheduled breakage as they are consumed, and
coordinates [advances](../advance/README.md). Billing decides the amount and
settlement policy; the collector commits the plan's group and creates billing
allocations from each consumption.

## Execution contract

A plan owns the postings and reservations for one customer's ledger group.
Each `Consume` returns a `Consumption` with its amounts, selected
sources, and transaction IDs. Reserved credit cannot be consumed again by later
operations. All FBO consumption in a ledger group must use the same plan.

`Finalize` ends planning and returns its postings together with any
caller-provided companion postings. Further consumption or finalization is
rejected. The caller keeps planning, breakage
bookkeeping, and group posting in one database transaction. A planning failure
requires the caller to roll back.

Consumption references resolve to the committed transactions in their original
planning order, independently of the group's transaction enumeration.

Breakage releases are recorded during planning and reference the exact consumption
entries, whose IDs are assigned before posting.

`SourceBalanceAsOf` is the historical cutoff for source selection. `BookedAt`
is the effective time of the consumption postings.

Sources are selected by priority, restricted before unrestricted credit, earliest
expiry, then stable source cursor. Receivable coverage never creates advance.
