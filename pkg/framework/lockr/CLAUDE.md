# lockr

<!-- archie:ai-start -->

> Two PostgreSQL advisory-lock implementations for distributed mutual exclusion: Locker (transaction-scoped, pg_advisory_xact_lock, requires active Ent tx in ctx) and SessionLocker (connection-scoped, pg_advisory_lock/unlock, requires a dedicated *sql.Conn). Keys are scoped strings hashed to uint64 via xxhash.

## Patterns

**LockForTX requires active Ent transaction in ctx** — Locker.LockForTX calls entutils.GetDriverFromContext(ctx); if no tx driver is found it returns an error. Callers must already be inside a transaction.RunWithNoValue or equivalent. (`transaction.RunWithNoValue(ctx, txCreator, func(ctx context.Context) error { return locker.LockForTX(ctx, key) })`)
**SessionLocker needs dedicated Postgres connection** — NewSessionLockr acquires conn via pgdriver.Driver.DB().Conn(ctx) at construction time. The connection must be closed via SessionLocker.Close(). (`locker, err := lockr.NewSessionLockr(ctx, SessionLockerConfig{Logger: l, PostgresDriver: pgdrv}); defer locker.Close()`)
**Key construction enforces non-empty, no-colon scopes** — NewKey(scopes...) rejects empty scopes and scopes containing ':' (the join separator). Hash64 uses xxhash for collision-resistance. (`key, err := lockr.NewKey("billing", customerID)`)
**SessionLocker Releaser is idempotent** — releaser.release uses sync.Mutex + done flag so calling the returned Releaser func twice is safe (second call is a no-op). (`rel, _ := locker.Lock(ctx, k); rel(ctx); rel(ctx) // second call is safe`)
**pg lock_timeout via pgdriver.WithLockTimeout** — ErrLockTimeout is returned when PostgreSQL error 55P03 is received; do not use context.WithTimeout for lock acquisition (pgx cancels the connection on context cancel). (`pgdriver.NewPostgresDriver(ctx, url, pgdriver.WithLockTimeout(3*time.Second))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `locker.go` | Transaction-scoped advisory lock via pg_advisory_xact_lock; lock released automatically on tx commit/rollback. | getTxClient checks that transaction_timestamp() != statement_timestamp() to verify a real tx is open; using autocommit connection will fail. |
| `session.go` | Session-scoped advisory lock; Lock blocks, TryLock is non-blocking, returns ErrNoLockAcquired if denied. | SessionLocker uses a single sql.Conn and a sync.Mutex; concurrent Lock calls on the same SessionLocker instance are serialized (not parallelized). |
| `key.go` | Key construction and xxhash hashing; scopes joined with ':' separator. | Scopes must not contain ':' — NewKey returns an error if they do. |
| `var.go` | PostgreSQL error code constant 55P03 (lock_timeout). | checkForTimeout does string-contains match on error message — may have false positives if error wrapping changes. |

## Anti-Patterns

- Calling Locker.LockForTX outside a Postgres transaction — returns error immediately
- Using context.WithTimeout for lock acquisition with Locker — pgx cancels the connection on ctx cancel, leaving tx in error state; use WithLockTimeout on the pgdriver instead
- Sharing a single SessionLocker instance across goroutines with high lock contention — the internal sync.Mutex serializes all Lock/TryLock calls
- Forgetting to call SessionLocker.Close() — leaks the dedicated sql.Conn
- Constructing Key scopes that contain ':' — NewKey returns an error

## Decisions

- **Two lock types for different lifetime requirements** — Transaction-scoped locks (Locker) auto-release on commit/rollback — ideal for billing charge advancement. Session-scoped locks (SessionLocker) persist across transactions — needed for long-running background jobs.
- **pg_advisory_xact_lock over application-level mutex** — PostgreSQL advisory locks are released atomically with the transaction, preventing stale locks after crashes. In-process mutexes don't survive process restarts or multi-instance deployments.

## Example: Acquire a transaction-scoped advisory lock for a customer

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
    // safe to mutate customer billing state
    return doWork(ctx)
})
```

<!-- archie:ai-end -->
