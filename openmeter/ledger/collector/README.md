# Credit Collection

This package turns customer credit and advance into accrued value. The hard part is not posting `FBO -> accrued`; it is preserving the order of what was collected so later correction and breakage flows can undo the same economic slices.

## Vocab

- `BookedAt`: timestamp used for the ledger transactions being written.
- `SourceBalanceAsOf`: timestamp used to decide which FBO sources are available.
- `source`: one spendable FBO slice selected by the collector.
- `allocation`: billing's collapsed record of collected credit.
- `source entry`: the concrete negative FBO ledger entry created by collection.

`BookedAt` and `SourceBalanceAsOf` are intentionally separate. A charge can be booked at one business timestamp while source selection uses the current view of available credit, including future-dated expiry entries that are already visible by then.

## Collection Order

FBO collection order is:

```text
credit_priority asc
expires_at asc
stable cursor asc
```

Non-expiring credit sorts after expiring credit with the same priority.

The collector must use the same order as breakage release. This is what lets breakage avoid grant-level lineage. If the collector consumes an expiring source, it also asks breakage to release the matching planned breakage for that same source.

## Why Source Entries Carry Identity

Billing allocations are intentionally coarser than ledger collection internals. For example, a single allocation can represent multiple FBO source entries.

That means corrections cannot rely only on the allocation amount. A partial correction needs to know the original internal order:

```text
usage collected 10:
  source #0: 4 from expiry E1
  source #1: 6 from expiry E2

correction restores 5:
  restore 5 from source #1
```

The source entry identity/order is the stable bridge between:

- the committed ledger entries,
- the breakage release attached to each expiring source,
- and the later correction request against the billing allocation.

The amount still comes from ledger entries. The identity/order only says which committed source entry came first.

## Forward Collection Example

Suppose the customer has:

```text
priority 0, expires E1: 7
priority 0, expires E2: 5
priority 1, no expiry: 9
```

Collecting 10 chooses:

```text
7 from E1
3 from E2
```

The ledger posts the FBO sources to accrued at `BookedAt` and breakage releases the same expiring slices at their expiry timestamps:

```text
@BookedAt
FBO(E1)  -7
FBO(E2)  -3
accrued +10

@E1 [release]
FBO      +7
breakage -7

@E2 [release]
FBO      +3
breakage -3
```

The non-expiring source is untouched because lower-priority expiring credit was enough.

## Advance

If credit-only collection cannot cover the full amount, the shortfall is advanced:

```text
@BookedAt
receivable -x
FBO        +x

@BookedAt
FBO     -x
accrued +x
```

Advance does not create breakage. It has no expiring real credit backing it yet.

If a later expiring credit purchase backfills that advance, breakage treats the covered amount as already used: it creates a plan and an immediate release for the same expiry.

## Corrections

Usage correction restores previously collected value. It does not increase usage; it unwinds up to the original collected amount.

For a partial correction, the collector restores source entries in reverse original collection order:

```text
original collection:
  source #0: 4 from E1
  source #1: 6 from E2

correction of 5:
  restore 5 from source #1
```

If source #1 had a breakage release, that release is reopened for 5. If the correction were 8, it would reopen:

```text
6 from source #1
2 from source #0
```

This reverse order is what keeps remaining usage equivalent to the original collection prefix:

```text
original:  E1(4), E2(6)
correct 5
remaining used: E1(4), E2(1)
```

## Backfilled Advance Correction

Backfilled advance is a two-time problem:

1. original usage consumed advance;
2. later real credit covered that already-used advance.

Correcting the original usage has to unwind both facts:

- undo the original advance-backed collection,
- unwind the later backfill attribution,
- reopen the advance-backfill breakage release,
- and make the covered real credit available again as ordinary FBO credit.

This is why correction follows the active lineage state first, then maps back to the original source entry. The source entry tells us what was collected; lineage tells us what that value became later.

## Transaction Boundary

Collection and correction must run inside one database transaction. Source selection, ledger commit, breakage record persistence, and billing realization/correction creation are one logical operation.

If any step commits independently, later flows can see impossible intermediate states: ledger entries without breakage records, breakage records without ledger entries, or billing allocations pointing at incomplete ledger work.
