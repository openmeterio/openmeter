# Credit Collection

This package turns selected customer FBO credit into accrued value and, for
custom-currency `credit_then_invoice` overage, fiat receivable coverage.
Credit-only accrual asks [advance](../advance/README.md) to create an advance for
an uncovered amount. The
hard part is preserving the exact order of selected sources so later correction
and breakage flows can undo the same economic slices.

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
feature-restricted before unrestricted
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

## Custom-Currency CTI Receivable Coverage

Custom-currency `credit_then_invoice` creates its gross fiat receivable before
asking the collector to cover part of it with eligible settlement-fiat FBO
credit. This path is not a general receivable settlement mechanism. It uses the
same priority, expiry, feature, and breakage-release rules as accrued
collection, but it never creates advance for an uncovered remainder.

Example:

```text
gross receivable:      RECEIVABLE -5
available fiat credit: FBO          +3

coverage:
FBO        -3
RECEIVABLE +3

remaining receivable: RECEIVABLE -2
```

Coverage preserves the original credit's source charge and records the charge
whose receivable is covered as the spend charge. Correction restores the exact
selected FBO sources and reopens their breakage releases. Each source slice has
a collection origin, but is not eligible for earnings recognition because
covering a receivable creates no accrued value.

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

Every new source slice also receives an immutable `collection_origin_id`. Its downstream
backfill, recognition, and correction entries retain that origin. Two runs of
the same spend charge consuming the same purchase therefore remain independently
correctable. Reused FBO credit starts a fresh origin.

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

[Correction](correction/README.md) restores collected value in reverse original
collection order. It owns provenance and legacy history handling, the shared
source-selection rules, and correction posting with breakage reopening.

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
