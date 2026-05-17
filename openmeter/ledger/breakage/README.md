# Credit Expiration Breakage

This package keeps future credit-expiration ledger entries aligned with actual customer credit usage.

The ledger is still the source of accounting truth. Breakage records are an allocation/index layer: they let later collection and correction flows find which future breakage entries are still open, which releases can still be reopened, and which expired amounts should be shown to customers.

## Vocab

- `plan`: future breakage for expiring credit that might remain unused.
- `release`: future inverse entry that reduces a plan because the credit was used before expiry.
- `reopen`: future entry that restores a released amount because a correction made that credit unused again.
- `breakage impact`: customer-visible expired amount after all plans, releases, and reopens are netted.

All amounts on breakage records are positive. The sign lives in the ledger entries.

## Core Invariant

Breakage can avoid grant-level lineage only if credit collection and breakage release use the same order:

```text
credit_priority asc
expires_at asc
stable cursor asc
```

`expires_at` is the planned breakage transaction's `booked_at`. It is not a route dimension.

This is the key correctness argument:

1. Expiring issued credit creates a future plan.
2. The collector consumes available FBO sources in the same order it asks breakage for open plans.
3. Every consumed planned slice creates a release against that plan.
4. Therefore the open planned amount at an expiry is exactly the remaining unused expiring credit for that expiry.

If collection order and release order diverge, a release can reduce the wrong expiry. At that point the system would need explicit grant lineage to recover correctness.

## Ledger Shapes

Issuing expiring credit creates normal FBO credit now and planned breakage at expiry:

```text
@T
FBO +10

@E [plan]
FBO      -10
breakage +10
```

Using part of that credit before expiry releases the planned breakage:

```text
@U
FBO     -4
accrued +4

@E [release]
FBO      +4
breakage -4
```

At `E`, the net FBO impact is `-6`, so the customer sees 6 expired credits.

If the usage is later corrected and 2 credits become unused again, the release is reopened:

```text
@C
FBO     +2
accrued -2

@E [reopen]
FBO      -2
breakage +2
```

At `E`, the net FBO impact is now `-8`.

## Advance Coverage

Advance-backed usage has no expiring real credit yet, so it creates no breakage.

When later expiring credit covers already-used advance, that credit is both issued and already consumed from the perspective of breakage:

```text
@E [plan]
FBO      -5
breakage +5

@E [release]
FBO      +5
breakage -5
```

The net breakage impact is zero because the covered credit was already used. If the original advance-backed usage is corrected later, the release is reopened and the covered credit can expire unused.

## Corrections

Corrections do not replay global collection. They unwind the original committed collection order.

Forward collection entries carry stable source identity/order metadata. A usage correction walks the original FBO source entries in reverse collection order and reopens the releases attached to those entries.

That reverse walk matters for partial corrections. If a usage consumed:

```text
5 from expiry E1
3 from expiry E2
```

then correcting 4 restores:

```text
3 from E2
1 from E1
```

This matches the intuitive stack behavior: the last consumed credit is the first restored credit.

## Expired Visibility

Expired customer transactions are not individual breakage records. They are the net impact of all visible breakage records grouped by expiry and currency:

```text
impact = -(plans - releases + reopens)
```

Zero-impact groups are hidden. Negative internal breakage totals are invalid because releases/reopens would exceed their backing plans.

The visible transaction cursor is the newest ledger transaction cursor among the records that contributed to the net impact. That keeps pagination stable while still presenting one customer-facing expired row per expiry/currency bucket.
