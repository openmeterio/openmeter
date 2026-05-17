# Credit Collection

This package turns customer credit and advance into accrued value. The hard part is not posting `FBO -> accrued`; it is preserving the exact order of what was collected so later correction and breakage flows can undo the same economic slices.

## Vocab

- `BookedAt`: timestamp used for the ledger transactions being written.
- `SourceBalanceAsOf`: timestamp used to decide which FBO sources are available.
- `source`: one spendable FBO slice selected by the collector.
- `source entry`: the concrete negative FBO ledger entry created by collection.
- `allocation`: billing's collapsed record of collected credit.
- `advance`: value moved through FBO/accrued before real credit exists to cover it.

`BookedAt` and `SourceBalanceAsOf` are intentionally separate.

Example:

```text
charge allocates at T1
source balance is checked at T5

BookedAt = T1
SourceBalanceAsOf = T5
```

The transaction is booked at `T1`, but source selection can see credit and expiry state visible as of `T5`.

## Collection Order

FBO collection order is:

```text
credit_priority asc
expires_at asc
stable cursor asc
```

Non-expiring credit sorts after expiring credit with the same priority.

This order must match breakage release order. If the collector consumes an expiring source, it also asks breakage to release the matching planned breakage for that same source.

## Forward Collection Example

Assume the customer has:

```text
source A: priority 0, expires T10, available 10
source B: priority 0, expires T15, available 15
source C: priority 1, no expiry,  available 20
```

Collecting 5 at `T2` chooses source A:

```text
@T2
FBO(A)  -5
ACCRUED +5
```

Breakage release for the selected expiring source:

```text
@T10 [release]
FBO(A) +5
BR     -5
```

Collecting another 10 at `T3` chooses the rest of source A, then source B:

```text
@T3
FBO(A)  -5
FBO(B)  -5
ACCRUED +10
```

Breakage releases:

```text
@T10 [release]
FBO(A) +5
BR     -5

@T15 [release]
FBO(B) +5
BR     -5
```

Source C is untouched because all lower-priority expiring credit was consumed first.

## Source Entry Identity

Billing allocations are intentionally coarser than ledger collection internals. A single allocation can represent multiple FBO source entries.

Example:

```text
allocation amount = 10

ledger source entries:
  source #0: FBO(A) -5
  source #1: FBO(B) -5
```

The ledger entries carry source identity/order metadata:

```text
source #0 -> order 0
source #1 -> order 1
```

That identity is not a second source of numeric truth. Amounts come from ledger entries. The identity only records the order in which committed source entries were selected.

This bridge is needed because later correction starts from a billing allocation, but breakage releases are attached to concrete FBO source entries.

## Credit-Only Advance

If credit-only collection cannot cover the requested amount, the shortfall becomes advance.

Example:

```text
customer has 10 real credit
usage needs 15
```

Real credit collection:

```text
@T
FBO(real) -10
ACCRUED   +10
```

Advance creation and collection:

```text
@T
RECEIVABLE -5
FBO        +5

@T
FBO(advance) -5
ACCRUED      +5
```

Advance does not create breakage because no expiring real credit backs it yet.

## Advance Backfill

When later real credit covers advance, the covered value is already used from the collector's perspective.

Example:

```text
T1 usage creates 5 advance
T5 expiring credit purchase covers that advance
T20 purchased credit expires
```

Breakage sees the covered amount as issued and immediately used:

```text
@T20 [plan]
FBO(real) -5
BR        +5

@T20 [release]
FBO(real) +5
BR        -5
```

Net breakage is zero unless the original advance-backed usage is later corrected.

## Usage Corrections

Usage correction restores previously collected value. It does not increase usage; it unwinds up to the original collected amount.

Correction uses reverse original collection order.

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

Backfilled advance is a two-time problem:

1. original usage consumed advance;
2. later real credit covered that already-used advance.

Correcting the original usage has to unwind both facts:

- undo the original advance-backed collection;
- unwind the later backfill attribution;
- reopen the advance-backfill breakage release;
- make the covered real credit available again as ordinary FBO credit.

Example:

```text
T1 usage consumes 5 advance
T5 credit purchase backfills that 5, expires T20
T6 original usage is corrected by 5
```

The correction:

```text
@T6
FBO(advance) +5
ACCRUED      -5
```

The backfilled credit is no longer used, so breakage reopens the release:

```text
@T20 [reopen]
FBO(real) -5
BR        +5
```

The covered real credit is re-issued into ordinary FBO state so it can be consumed later or expire at `T20`.

## Transaction Boundary

Collection and correction must run inside one database transaction.

The atomic unit includes:

- source selection;
- ledger commit;
- breakage record persistence;
- billing allocation/correction creation.

If any step commits independently, later flows can observe impossible intermediate states:

```text
ledger entries without breakage records
breakage records without ledger entries
billing allocations pointing at incomplete ledger work
```
