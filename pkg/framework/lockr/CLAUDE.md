# lockr

<!-- archie:ai-start -->

> Two PostgreSQL advisory-lock implementations for distributed mutual exclusion: Locker (transaction-scoped pg_advisory_xact_lock, auto-released on tx commit/rollback) and SessionLocker (connection-scoped pg_advisory_lock, requires explicit Close). Keys are scoped strings hashed to uint64 via xxhash.

## Patterns

**LockForTX requires active Ent transaction in ctx** — Locker.LockForTX calls entutils.GetDriverFromContext(ctx) and fails if no tx driver is present. Call inside transaction.RunWithNoValue or entutils.TransactingRepo. It also verifies transaction_timestamp() != statement_timestamp() to confirm a real tx is open. (`return transaction.RunWithNoValue(ctx, txCreator, func(ctx context.Context) error { return locker.LockForTX(ctx, key) })`)
**SessionLocker requires Start() then Close()** — NewSessionLockr creates the struct; Start(ctx) acquires the dedicated sql.Conn from pgdriver.Driver.DB(). Always defer Close() — it closes the dedicated connection and releases all session-level locks. (`locker, _ := lockr.NewSessionLockr(SessionLockerConfig{Logger: l, PostgresDriver: pgdrv}); _ = locker.Start(ctx); defer locker.Close()`)
**Key construction enforces non-empty, no-colon scopes** — NewKey(scopes...) rejects empty scopes and scopes containing ':' (the join separator). Hash64 uses xxhash. Prefer charges.NewLockKeyForCharge or billing.WithLock helpers over raw lockr.NewKey. (`key, err := lockr.NewKey("billing", customerID) // scopes must not contain ':'`)
**Releaser is idempotent via sync.Mutex + done flag** — The Releaser returned by Lock/TryLock wraps a struct with sync.Mutex + done bool; calling it twice is a no-op. PostgreSQL session-level locks are reentrant — each Lock increments a refcount needing a matching unlock. (`rel, _ := locker.Lock(ctx, k); rel(ctx); rel(ctx) // second call is a no-op`)
**Use pgdriver.WithLockTimeout — not context.WithTimeout** — context.WithTimeout makes pgx cancel the connection on deadline, leaving the tx in error state. WithLockTimeout sets lock_timeout in RuntimeParams so PostgreSQL returns 55P03 (ErrLockTimeout) without destroying the connection. (`pgdriver.NewPostgresDriver(ctx, url, pgdriver.WithLockTimeout(3*time.Second))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `locker.go` | Transaction-scoped advisory lock via pg_advisory_xact_lock; released automatically on tx commit/rollback. | getTxClient verifies transaction_timestamp() != statement_timestamp() — an autocommit connection fails this check even with a tx driver in ctx. |
| `session.go` | Session-scoped advisory lock; Lock blocks, TryLock is non-blocking (ErrNoLockAcquired if denied), TryLock returns ErrSessionLockerBusy if the internal mutex is held. | Holds a single sql.Conn and a sync.Mutex; concurrent Lock calls on the same instance are serialized. Do not share one SessionLocker across goroutines under high contention. |
| `key.go` | Key construction and xxhash hashing; scopes joined with ':' separator. | Scopes must not contain ':' — NewKey errors. Hash collisions are possible but extremely unlikely. |
| `var.go` | PostgreSQL error code constant 55P03 (lock_timeout). | checkForTimeout does a string-contains match on the error message — possible false positives if wrapping includes the literal '55P03'. |

## Anti-Patterns

- Calling Locker.LockForTX outside a Postgres transaction — returns an error immediately from getTxClient.
- Using context.WithTimeout for lock acquisition — pgx cancels the connection on ctx cancel; use pgdriver.WithLockTimeout instead.
- Sharing a single SessionLocker instance across goroutines with high lock contention — the internal mutex serializes all Lock/TryLock calls.
- Forgetting to call SessionLocker.Close() — leaks the dedicated sql.Conn from the pool.
- Hand-constructing lockr.Key strings instead of using charges.NewLockKeyForCharge or billing.WithLock helpers.

## Decisions

- **Two lock types for different lifetime requirements.** — Transaction-scoped locks (Locker) auto-release on commit/rollback — ideal for billing charge advancement. Session-scoped locks (SessionLocker) persist across transactions — needed for long-running admin jobs spanning multiple DB transactions.
- **pg_advisory_xact_lock over an application-level mutex.** — Advisory locks release atomically with the transaction, preventing stale locks after crashes; in-process mutexes don't survive multi-instance deployments.

## Example: Acquire a transaction-scoped advisory lock for a customer before billing mutation

```
import (
    "github.com/openmeterio/openmeter/pkg/framework/lockr"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

key, err := lockr.NewKey("billing", customerID)
if err != nil { return err }
return transaction.RunWithNoValue(ctx, txCreator, func(ctx context.Context) error {
    if err := locker.LockForTX(ctx, key); err != nil { return err }
    return doWork(ctx)
})
```

<!-- archie:ai-end -->
