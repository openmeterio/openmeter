# lockr

<!-- archie:ai-start -->

> Two PostgreSQL advisory-lock implementations for distributed mutual exclusion: Locker (transaction-scoped pg_advisory_xact_lock, auto-released on tx commit/rollback) and SessionLocker (connection-scoped pg_advisory_lock, requires explicit Close). Keys are scoped strings hashed to uint64 via xxhash.

## Patterns

**LockForTX requires active Ent transaction in ctx** — Locker.LockForTX calls entutils.GetDriverFromContext(ctx) and fails if no tx driver is present. Always call inside transaction.RunWithNoValue or entutils.TransactingRepo. It also verifies transaction_timestamp() != statement_timestamp() to confirm a real tx is open. (`return transaction.RunWithNoValue(ctx, txCreator, func(ctx context.Context) error { return locker.LockForTX(ctx, key) })`)
**SessionLocker requires Start() then Close()** — NewSessionLockr creates the struct; Start(ctx) acquires the dedicated sql.Conn from pgdriver.Driver.DB(). Always defer Close() — it calls pool.Close() on the dedicated connection and releases all session-level locks. (`locker, _ := lockr.NewSessionLockr(SessionLockerConfig{Logger: l, PostgresDriver: pgdrv}); _ = locker.Start(ctx); defer locker.Close()`)
**Key construction enforces non-empty, no-colon scopes** — NewKey(scopes...) rejects empty scopes and scopes containing ':' (the join separator). Hash64 uses xxhash for collision-resistance. Use charges.NewLockKeyForCharge or billing.WithLock helpers rather than raw lockr.NewKey. (`key, err := lockr.NewKey("billing", customerID) // scopes must not contain ':'`)
**Releaser is idempotent via sync.Mutex + done flag** — The Releaser func returned by Lock/TryLock wraps a releaser struct with sync.Mutex + done bool. Calling it twice is safe — second call is a no-op. PostgreSQL session-level locks are reentrant; each Lock increments a refcount requiring a matching unlock. (`rel, _ := locker.Lock(ctx, k); rel(ctx); rel(ctx) // second call is a no-op`)
**Use pgdriver.WithLockTimeout — not context.WithTimeout** — context.WithTimeout causes pgx to cancel the connection on deadline, leaving the tx in an error state. WithLockTimeout sets lock_timeout in RuntimeParams, letting PostgreSQL return error code 55P03 (ErrLockTimeout) without destroying the connection. (`pgdriver.NewPostgresDriver(ctx, url, pgdriver.WithLockTimeout(3*time.Second))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `locker.go` | Transaction-scoped advisory lock via pg_advisory_xact_lock; lock is released automatically on tx commit/rollback. | getTxClient verifies transaction_timestamp() != statement_timestamp() — using an autocommit connection will fail this check even if a tx driver is in ctx. |
| `session.go` | Session-scoped advisory lock; Lock blocks, TryLock is non-blocking (returns ErrNoLockAcquired if denied), TryLock returns ErrSessionLockerBusy if the internal mutex is held. | SessionLocker holds a single sql.Conn and a sync.Mutex; concurrent Lock calls on the same instance are serialized. Do not share one SessionLocker across goroutines under high contention. |
| `key.go` | Key construction and xxhash hashing; scopes joined with ':' separator. | Scopes must not contain ':' — NewKey returns an error. Hash collisions are possible but extremely unlikely; two different scope strings can map to the same uint64. |
| `var.go` | PostgreSQL error code constant 55P03 (lock_timeout). | checkForTimeout does string-contains match on the error message — may have false positives if error wrapping includes the literal '55P03'. |

## Anti-Patterns

- Calling Locker.LockForTX outside a Postgres transaction — returns error immediately from getTxClient
- Using context.WithTimeout for lock acquisition — pgx cancels the connection on ctx cancel, leaving the tx in an error state; use pgdriver.WithLockTimeout instead
- Sharing a single SessionLocker instance across goroutines with high lock contention — the internal sync.Mutex serializes all Lock/TryLock calls
- Forgetting to call SessionLocker.Close() — leaks the dedicated sql.Conn from the pool
- Hand-constructing lockr.Key strings instead of using charges.NewLockKeyForCharge or billing.WithLock helpers

## Decisions

- **Two lock types for different lifetime requirements** — Transaction-scoped locks (Locker) auto-release on commit/rollback — ideal for billing charge advancement. Session-scoped locks (SessionLocker) persist across transactions — needed for long-running admin jobs that span multiple DB transactions.
- **pg_advisory_xact_lock over application-level mutex** — Advisory locks are released atomically with the transaction, preventing stale locks after crashes or process restarts. In-process mutexes don't survive multi-instance deployments.

## Example: Acquire a transaction-scoped advisory lock for a customer before billing mutation

```
import (
    "github.com/openmeterio/openmeter/pkg/framework/lockr"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

key, err := lockr.NewKey("billing", customerID)
if err != nil { return err }

return transaction.RunWithNoValue(ctx, txCreator, func(ctx context.Context) error {
    if err := locker.LockForTX(ctx, key); err != nil {
        return err
    }
    // safe to mutate customer billing state inside this transaction
    return doWork(ctx)
})
```

<!-- archie:ai-end -->
