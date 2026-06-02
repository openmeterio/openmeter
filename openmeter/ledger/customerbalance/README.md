# Customer Credit Balance

This package exposes customer-facing credit balance and credit transaction views.

> The current balance calculation path is temporary. It is expected to move to the **real-time-engine (RTE)** shortly. The API semantics below should remain stable even if the implementation stops querying the ledger directly for every balance view.

## Balance Semantics

Credit balance is defined at a point in time.

`asOf` controls which booked FBO ledger entries are visible:

```text
balance(asOf=T) = sum(FBO entries where booked_at <= T)
```

Future-dated expiration entries do not affect the current balance. They do affect a balance queried at or after their expiration timestamp.

Example:

```text
@T1
FBO +10

@T10 [breakage.plan]
FBO -10
BR  +10
```

Balances:

```text
asOf T5  => 10
asOf T10 => 0
```

If 4 credits are used before expiry:

```text
@T5
FBO     -4
ACCRUED +4

@T10 [breakage.release]
FBO +4
BR  -4
```

Balances:

```text
asOf T4  => 10
asOf T5  => 6
asOf T10 => 0
```

The `T10` balance is zero because the remaining 6 expired at `T10`.

## Transaction Listing

Customer-visible credit transactions are a read model over ledger and billing activity. They are not a raw dump of ledger transactions.

Visible types:

- `funded`: credit became available.
- `consumed`: credit was used.
- `expired`: unused credit expired.

The visible amount is the customer FBO impact:

```text
funded   => positive
consumed => negative
expired  => negative
```

## Listing Example: Funded, Consumed, Expired

Credit issuance:

```text
@T1
FBO +10
```

Usage:

```text
@T5
FBO     -4
ACCRUED +4
```

Expiration:

```text
@T10 [plan]
FBO -10
BR  +10

@T10 [release]
FBO +4
BR  -4
```

Customer-visible transaction listing as of `T10`:

```text
T10 expired  -6
T5  consumed -4
T1  funded   +10
```

Listing as of `T5`:

```text
T5 consumed -4
T1 funded   +10
```

The expired row is hidden before `T10` because the breakage entries are future-dated.

## Expired Credit Projection

Expired rows come from breakage impacts, not from raw breakage ledger transactions.

Breakage can have multiple records at the same expiry:

```text
@T10 [plan]    10
@T10 [release] 4
@T10 [reopen]  2
```

Customerbalance should show one net expired row:

```text
plans - releases + reopens = 10 - 4 + 2 = 8
expired amount = -8
```

Zero-impact groups are hidden:

```text
@T20 [plan]    5
@T20 [release] 5

expired amount = 0
```

This is common for expiring credit that immediately backfilled already-used advance.

## Cursor Semantics

Transaction listing is ordered by ledger cursor:

```text
booked_at
created_at
transaction_id
```

For expired rows, customerbalance uses the cursor from the newest breakage transaction that contributed to the visible net impact.

Example:

```text
@T10 [plan]    created at C1
@T10 [release] created at C2
```

The visible expired row is booked at `T10` and uses the `C2` cursor. This gives one stable customer-facing row for the expiry bucket while preserving cursor pagination.

## Type Filtering

If the caller requests a specific type:

```text
type=funded
```

only funded rows are returned.

If the caller requests:

```text
type=expired
asOf=T10
```

only expired rows visible at `T10` are returned.

The `asOf` boundary applies before projection. Future breakage entries beyond `asOf` must not leak into the listing.

## Presentation Boundary

This package may:

- merge funded, consumed, and expired views;
- apply customer-facing labels and balances;
- page and cursor the result set;
- expose FBO impact as customer-visible amount.

This package should not:

- decide how plan/release/reopen rows net;
- decide FBO collection order;
- decide correction unwind order.

Those correctness rules belong to `breakage` and `collector`.
