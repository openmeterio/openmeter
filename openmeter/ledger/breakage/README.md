# Credit Expiration Breakage

This package keeps future credit-expiration ledger entries aligned with actual customer credit usage.

The ledger is the accounting source of truth. Breakage records are an allocation/index layer: they let later collection and correction flows find open planned breakage, reopen released breakage, and project customer-visible expired credit.

## Notation

- `FBO(r)` is the customer credit account route.
- `BR(b)` is the breakage account route used for breakage accounting.
- `r.priority` is the credit draw-down priority. Lower values are consumed first.
- `@T` is ledger `booked_at`.
- `E` is an expiration timestamp.
- `plan` means pre-booked future breakage for issued expiring credit.
- `release` means a future entry that reduces planned breakage because credit was used or removed before expiration.
- `reopen` means a future entry that increases planned breakage again because previously used credit became unused.
- `breakage impact` means the customer-visible expired amount after plans, releases, and reopens are netted.

All breakage record amounts are positive. The sign lives in the ledger entries.

## Core Invariant

Breakage avoids grant-level lineage only if:

```text
FBO consumption order == breakage release order
```

The shared ordering is:

```text
credit_priority asc
expires_at asc
stable cursor asc
```

`credit_priority` comes from the FBO route. `expires_at` is the planned breakage transaction's `booked_at`; it is not copied into the FBO route.

The correctness argument:

1. Expiring issued credit creates a future `plan`.
2. The collector consumes available FBO sources in the same order it asks breakage for open plans.
3. Every consumed planned slice creates a `release` against that same plan.
4. Therefore the open planned amount at an expiry is exactly the remaining unused expiring credit for that expiry.

If collection order and release order diverge, a release can reduce the wrong expiry. At that point the system would need explicit grant lineage to recover correctness.

## Routing And Metadata

Use route dimensions for fields that define collection eligibility, ordering, or routing validation.

Use annotations or breakage records for metadata and links.

That means:

- `credit_priority` is a route dimension because it affects normal FBO collection.
- `expires_at` is represented by planned breakage `booked_at`.
- breakage kind and source links are annotations/record data.
- breakage account routes should carry only dimensions relevant to breakage accounting/revenue recognition.

Breakage-generated FBO entries must not be treated as normal credit issuance or usage. They are marked as breakage activity:

```text
ledger.collection.type = breakage
ledger.breakage.kind = plan|release|reopen
```

## Type: Plan

When expiring credit is issued:

```text
@T
FBO(route{priority=P}): +x
OFFSET:                 -x
```

If the credit has no expiration, breakage does nothing.

If the credit expires at `E`, breakage pre-books a plan:

```text
@E [breakage.plan]
FBO(route{priority=P}): -x
BR(b):                  +x
```

Record shape:

```text
type = plan
amount = x
expires_at = E
fbo_route = route
breakage_route = b
source_transaction = <credit issuance tx>
breakage_transaction = <planned breakage tx>
```

Example:

```text
10 credit arrives at T1, expires at T10

@T1
FBO +10

@T10 [plan]
FBO -10
BR  +10
```

If nothing else happens, the balance is 10 before `T10` and 0 at `T10`.

## Type: Release

When usage consumes real credit:

```text
@T
FBO(route):     -x
ACCRUED/OFFSET: +x
```

Breakage walks open plans in the same order as FBO collection:

```text
credit_priority asc, expires_at asc, stable cursor asc
```

For each selected plan slice `y`, it books a release at that plan's expiry:

```text
@plan.expires_at [breakage.release]
FBO(plan.fbo_route):     +y
BR(plan.breakage_route): -y
```

Record shape:

```text
type = release
amount = y
plan = <breakage.plan>
source_transaction = <FBO consumption tx>
source_entry = <FBO source entry, when available>
breakage_transaction = <future release tx>
```

Example:

```text
10 credit at T1, expires T10
15 credit at T5, expires T15
```

Initial plans:

```text
@T10 [plan]
FBO -10
BR  +10

@T15 [plan]
FBO -15
BR  +15
```

Usage of 5 at `T2` consumes the first expiring plan:

```text
@T2
FBO     -5
ACCRUED +5

@T10 [release]
FBO +5
BR  -5
```

Usage of 10 at `T3` continues from the remaining `T10` plan, then moves to `T15`:

```text
@T3
FBO     -10
ACCRUED +10

@T10 [release]
FBO +5
BR  -5

@T15 [release]
FBO +5
BR  -5
```

The releases follow expiry order without needing to know which "grant" the usage came from.

## Type: Reopen

Usage correction restores previously consumed credit:

```text
@T
FBO(route):     +x
ACCRUED/OFFSET: -x
```

Breakage reopens releases produced by the original consumption in reverse unwind order:

```text
@plan.expires_at [breakage.reopen]
FBO(plan.fbo_route):     -y
BR(plan.breakage_route): +y
```

This increases breakage because the credit is unused again.

Example:

```text
original collection:
  source #0: 5 from expiry T10
  source #1: 5 from expiry T15

correction restores 7
```

Breakage reopens:

```text
@T15 [reopen]
FBO -5
BR  +5

@T10 [reopen]
FBO -2
BR  +2
```

The remaining usage is equivalent to the original collection prefix:

```text
T10 used: 3
T15 used: 0
```

## Usage Corrections

Usage corrections are corrections of prior `FBO -> accrued` movement.

Usage correction is an unwind up to the original collected amount. It restores FBO and reopens any breakage releases attached to the restored source entries.

If a future flow ever produces additional real-credit consumption as a correction, it should apply the normal release rule:

```text
@T
FBO(route):     -x
ACCRUED/OFFSET: +x

@plan.expires_at [breakage.release]
FBO(plan.fbo_route):     +y
BR(plan.breakage_route): -y
```

The currently important path is restoration:

```text
@T
FBO(route):     +x
ACCRUED/OFFSET: -x

@plan.expires_at [breakage.reopen]
FBO(plan.fbo_route):     -y
BR(plan.breakage_route): +y
```

## Credit Purchase Corrections

Credit purchase corrections would correct prior `receivable -> FBO` issuance.

Removing issued expiring credit would reduce planned breakage by booking a release at the matching expiry:

```text
@T
FBO(route):       -x
RECEIVABLE/OTHER: +x

@plan.expires_at [breakage.release]
FBO(plan.fbo_route):     +y
BR(plan.breakage_route): -y
```

Restoring issued expiring credit would behave like expiring credit issuance and create a plan.

This source kind is reserved, but full credit-purchase correction semantics are not implemented yet. The charge domain still needs to define correction/delete policy, removable amount checks, and behavior when the purchased credit has already been consumed.

## Advance

Advance-backed usage has no expiring real credit yet, so it creates no breakage.

```text
@T
FBO(advance/no-expiry): -x
ACCRUED/OFFSET:         +x
```

Example:

```text
usage needs 15, but only 10 real credit exists

@T
FBO(real)    -10
ACCRUED      +10

@T
FBO(advance) -5
ACCRUED      +5
```

Only the real-credit portion can release existing planned breakage. The advance portion has no breakage until real expiring credit later covers it.

## Advance Backfill

Advance backfill means later real credit is assigned to already-used advance-backed value.

For breakage, this is both:

1. issuance of expiring real credit;
2. immediate consumption of that same credit, because the advance usage already happened.

If 5 advance is covered by new credit at `T5` that expires at `T20`, the FBO attribution makes the covered value real credit and breakage books both sides:

```text
@T20 [plan]
FBO(route{priority=P}): -5
BR(b):                  +5

@T20 [release]
FBO(route{priority=P}): +5
BR(b):                  -5
```

Net breakage for the covered advance is zero.

If the original advance-backed usage is corrected later, the advance-backfill release is reopened:

```text
@T20 [reopen]
FBO(route{priority=P}): -5
BR(b):                  +5
```

Now the covered credit is unused again and can expire.

## Expired Breakage Impact

Customer-visible expired credit is not one breakage transaction. It is the net impact of visible breakage records grouped by expiry and currency:

```text
impact = -(plans - releases + reopens)
```

Zero-impact groups are hidden. Negative internal breakage totals are invalid because releases/reopens exceeded their backing plans.

Example:

```text
@T10 [plan]    10
@T10 [release] 5
@T10 [reopen]  2

plans - releases + reopens = 10 - 5 + 2 = 7
customer-visible impact = -7
```

The visible transaction cursor is the newest ledger transaction cursor among the records that contributed to the net impact. That keeps pagination stable while presenting one customer-facing expired row per expiry/currency bucket.
