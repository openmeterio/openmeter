# Calculation demand

Demand records which changes RTE should track for a balance and whether a worker should act on
them in the background. The registration belongs to the key. Callers sharing a key also share
its registration, so one caller's `Deactivate` ends tracking for all of them. The last
successful balance stays available through `Get`.

| Operation | When the key gets recalculated |
| --- | --- |
| `Activate` | The worker picks up outstanding work and refreshes the balance after tracked changes. Activation persists until deactivation or source retirement. |
| `EnsureFresh` | The caller waits if the key is dirty, tracking lapsed, or the last calculation reached the caller's maximum age. A clean result can be reused while tracking remains active. |

For example, a caller can read every ten seconds with `PullTTL=1m` and `MaxAge=30s`.
Two usage batches between reads can share one refresh. If background tracking is off and the
caller returns after the lease expired, RTE refreshes even if no dirty event was recorded during
the gap. A zero maximum age allows any age while the key is continuously tracked and clean.

## Claims and queue order

A key is dirty while `dirty_version > completed_version`. An invalidation increments the dirty
version for matching active keys. It locks demand rows in ascending primary-key order so
overlapping invalidations acquire shared rows in the same order. If at least one key matched,
it stores a receipt for the CloudEvents source and ID in the same transaction. Duplicate delivery
then leaves those keys alone.
A distinct event for earlier usage still counts. A selector containing only namespace and kind
matches every active key of that kind.

The first change to a clean key makes it eligible for work. Further changes while it waits keep
its place. Background slots claim eligible rows in `next_attempt_at`, then ID order. Each takes
a separate calculation lock **before** reading the version it will cover. Other slots skip locked
keys. An `EnsureFresh` for that key waits for the lock and rechecks whether it needs to calculate.

Suppose a slot claims version 7 and two events advance the dirty version to 9 while it calculates.
The events can update the demand row while the slot holds the calculation lock. On success the
slot commits the retained state, any update outbox entry, and completion of **7**. The next pass
covers 9 and joins the queue at completion time, behind keys already waiting. Explicit one-time
calculations run independently of demand.

## Failure and expiry

- A crash rolls back the open transaction and PostgreSQL releases its lock. A calculation error
  rolls back the attempt's writes and schedules a retry with bounded exponential backoff.
  New changes advance the dirty version but preserve that retry deadline. A targeted
  `EnsureFresh` can retry immediately.
- Each demand calculation has a timeout. If cancellation closes the connection, the worker
  records backoff in a new transaction only if the versions and attempt count still match.
  When a caller owns the outer transaction it receives the error and must roll back.
- A metered entitlement's scheduled deletion is a known deadline. An activated key is checked
  when deletion takes effect. A key tracked only between reads is checked on the next refresh.
  Retirement ends tracking, preserves the stored value, and returns not-found to a targeted refresh.
- Receipts expire after seven days by default. Each instance prunes up to 1,000 at a time,
  normally once a minute, retrying a full batch after one second. Redelivery after expiry can
  trigger another calculation.
