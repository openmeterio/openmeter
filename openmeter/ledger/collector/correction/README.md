# Collection Correction

Usage correction restores previously collected value. It does not increase usage; it unwinds up to the original collected amount.

Correction uses reverse original collection order within the allocation's FBO
subaccount. Repeated corrections subtract prior immutable correction links before
selecting the next source slice. Breakage reopening uses that same selection.

If a shared earnings-recognition group contains other spends or cost bases,
correction reverses only the accrued route and source/spend provenance needed by
the original collection or backfill unwind. Group membership alone is not
sufficient to select recognized value.

One planner selects sources and amounts for both storage formats. It selects
sources in reverse original collection order and backing in reverse original
backing order, then unwinds recognition within the selected source. Recognition
batching cannot change which funding is returned. Backed advance remains ahead
of its uncovered remainder.

The provenance reader derives remaining accrued, earnings and receivable positions
from ledger balances scoped to `collection_origin_id` and source. It validates
whole-origin conservation per currency under posting locks. Original entries
supply immutable ordering, routes and exact correction references; template
history is not replayed to determine the current state.

The [legacy reader](../../../billing/charges/legacylineage/README.md) adapts active
segments and their original ledger references to the same planner. The legacy
writer persists its exact selected segment IDs and amounts; stale selections
abort the enclosing transaction. There is no second amount allocation during
persistence.

Batch preparation chooses the correction path from the original ledger entries.
Legacy preparation owns source reservations, template merging, and segment-selection
annotations; provenance preparation owns origin history and exact-entry reversals.
Both produce postings, pending breakage records, and billing correction records.
The shared flow locks, prepares the batch, commits, persists breakage, and attaches
the committed group reference. Mixed batches preserve the request's realization
order and resolve legacy templates after collecting all selections.

Example:

```text
original allocation amount = 10

source #0: 4 from expiry T10
source #1: 6 from expiry T15
```

Correction of 5 restores:

```text
5 from source #1
```

Ledger correction:

```text
@C
FBO(source #1) +5
ACCRUED        -5
```

Breakage correction:

```text
@T15 [reopen]
FBO(source #1) -5
BR             +5
```

Correction of 8 restores:

```text
6 from source #1
2 from source #0
```

Ledger correction:

```text
@C
FBO(source #1) +6
FBO(source #0) +2
ACCRUED        -8
```

Breakage correction:

```text
@T15 [reopen]
FBO(source #1) -6
BR             +6

@T10 [reopen]
FBO(source #0) -2
BR             +2
```

The remaining usage is equivalent to the original collection prefix:

```text
original:  T10(4), T15(6)
correct 8
remaining used: T10(2), T15(0)
```

## Backfilled Advance Correction

Correction owns source selection and earnings unwinding. The injected advance
service builds the advance and backfill reversals, purchased-credit restoration,
and associated breakage reopening. The correction service commits and persists the combined
plan under the same posting locks. Legacy template corrections remain deferred
until the legacy path merges amounts targeting the same original transaction.

Backfilled advance is a two-time problem:

1. original usage consumed advance;
2. later real credit covered that already-used advance.

Correcting the original usage has to unwind both facts:

- undo the original advance-backed collection;
- unwind the later backfill attribution;
- reopen the advance-backfill breakage release;
- make the covered real credit available again as ordinary FBO credit.

For two advances A then B, each for 20, a purchase of 25 backs A by 20 and B
by 5. After recognizing the purchased backing:

| Amount | Before correcting B | After correcting B by 7 |
| --- | ---: | ---: |
| A earnings | 20 | 20 |
| B earnings | 5 | 0 |
| B uncovered accrued | 15 | 13 |
| B uncovered receivable obligation | 15 | 13 |
| Available purchased FBO credit | 0 | 5 |

The shared selection rule chooses B's 5 funded units before 2 uncovered units.
The collector correction reverses the 5 of earnings. Advance correction unwinds that backing,
restores 5 of purchased credit, and cancels 7 of B's original collection and
receivable issue. For expiring purchased credit, it also reopens the corresponding
breakage release. A later correction of 3 from B cancels uncovered advance only;
it does not return the same purchased credit again.
