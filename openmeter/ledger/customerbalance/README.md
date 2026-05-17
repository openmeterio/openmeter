# Customer Credit Balance

This package exposes customer-facing credit balance and credit transaction views.

> The current balance calculation path is temporary. It is expected to move to the **RTE** shortly. The API semantics below should remain stable even if the implementation stops querying the ledger directly for every balance view.

## Balance Semantics

Credit balance is defined at a point in time.

`asOf` controls which booked ledger entries are visible:

```text
balance(asOf=T) = sum(FBO entries booked_at <= T)
```

That means future-dated expiration entries do not affect the current balance, but they do affect a balance queried at or after their expiration timestamp.

Example:

```text
@T1
FBO +10

@T10
FBO -10   // planned breakage
```

The balance is:

```text
asOf T5  => 10
asOf T10 => 0
```

## Transaction Listing

Customer-visible credit transactions are a read model over ledger and billing activity. They are not a raw dump of ledger transactions.

The visible transaction types are:

- `funded`: credit became available.
- `consumed`: credit was used.
- `expired`: unused credit expired.

Expired rows come from breakage impacts, not from raw breakage ledger transactions. Breakage can have multiple plan/release/reopen records at the same expiry; customerbalance should show the net customer-facing effect.

## Expired Credit Visibility

Expired credit appears only when it is visible at the query time.

```text
@T1
FBO +10

@T10 [plan]
FBO      -10
breakage +10

@T5 usage
FBO -4

@T10 [release]
FBO      +4
breakage -4
```

Listing as of `T5` should not show an expired row.

Listing as of `T10` should show:

```text
expired -6
```

The visible expired amount is the FBO impact:

```text
-(plans - releases + reopens)
```

## Cursor Semantics

Transaction listing is ordered by ledger cursor:

```text
booked_at
created_at
transaction_id
```

For expired rows, the cursor is derived from the newest breakage transaction that contributed to the visible net impact. That gives one stable customer-facing row for an expiry bucket while still allowing cursor pagination to work with ledger-backed ordering.

## Presentation Boundary

This package should not decide breakage correctness.

It may:

- merge funded, consumed, and expired views;
- apply customer-facing labels and balances;
- page and cursor the result set.

It should not:

- know how plan/release/reopen rows net;
- decide FBO collection order;
- decide correction unwind order.

Those rules belong to `breakage` and `collector`.
