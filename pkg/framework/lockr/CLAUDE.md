# lockr

<!-- archie:ai-start -->

> Distributed business-level locking on top of PostgreSQL advisory locks, with two implementations: Locker for transaction-scoped pg_advisory_xact_lock (auto-released at tx end) and SessionLocker for connection-scoped pg_advisory_lock/pg_try_advisory_lock with explicit Releaser callbacks.

## Patterns

**Key from scopes with hash64 keyspace** — Locks are keyed by lockr.NewKey(scopes...) where each scope is non-empty and may not contain ':'; the Key.String() joins on ':' and Hash64() xxh3-hashes it into the int64 advisory-lock keyspace. (`key, err := lockr.NewKey("subscription", subID)`)
**Transaction-bound locking via context driver** — Locker.LockForTX requires an active tx in ctx: getTxClient calls entutils.GetDriverFromContext(ctx), rebuilds a *db.Tx from its raw config, and verifies transaction_timestamp() != statement_timestamp() before issuing pg_advisory_xact_lock. Outside a tx it errors. (`tx, err := entutils.GetDriverFromContext(ctx); // 'lockr only works in a transaction'`)
**PG-side lock timeout, not context timeout** — Acquisition timeouts are enforced by the Postgres lock_timeout (set via pgdriver.WithLockTimeout); checkForTimeout maps SQLSTATE 55P03 (pgLockTimeoutErrCode) to ErrLockTimeout. Context-based cancellation is deliberately avoided because it errors the pgx tx. (`if strings.Contains(err.Error(), pgLockTimeoutErrCode) { return ErrLockTimeout }`)
**Session lock + idempotent Releaser** — SessionLocker holds a dedicated *sql.Conn (Start) and returns a Releaser closure per acquisition; the releaser is sync-guarded (mu + done) so calling it twice is a no-op, and PG session advisory locks are reentrant (counter-based, one unlock per lock). (`releaser, err := locker.Lock(ctx, k); defer releaser(ctx)`)
**Constructor config validation** — NewLocker validates LockerConfig.Logger; NewSessionLockr requires both Logger and PostgresDriver and tags the logger with a unique id. (`if c.Logger == nil { return fmt.Errorf("logger is required") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `locker.go` | Transaction-scoped Locker: LockForTX / LockForTXWithScopes issuing pg_advisory_xact_lock; getTxClient asserts an active pg transaction in ctx. | Only works inside a transaction.Driver-managed tx; returns an error if no driver is in ctx. Locks auto-release at tx commit/rollback — never released manually. |
| `session.go` | Connection-scoped SessionLocker with Lock/TryLock(+WithScopes), Releaser closures, Start (acquires dedicated conn), and Close (releases all + closes conn). | Must call Start before locking and Close when done or session locks dangle on the connection. TryLock returns ErrSessionLockerBusy if another goroutine holds the internal mutex, distinct from ErrNoLockAcquired. |
| `key.go` | Key interface + NewKey scope validation + xxh3 Hash64. | Empty scopes or scopes containing ':' are rejected; collisions are possible since 64-bit hash collapses the keyspace. |
| `var.go` | pgLockTimeoutErrCode = '55P03' (lock_not_available SQLSTATE). | Timeout detection is string-matched against this code; changing it silently breaks ErrLockTimeout mapping. |

## Anti-Patterns

- Using context.WithTimeout to bound lock acquisition — pgx cancellation errors the transaction; rely on PG lock_timeout instead.
- Calling Locker.LockForTX outside an active transaction.Driver transaction (it errors).
- Using a SessionLocker without Start, or forgetting Close, leaving advisory locks dangling on the dedicated connection.
- Constructing Locker/SessionLocker directly instead of via NewLocker/NewSessionLockr (skips required-dependency validation).

## Decisions

- **Two distinct lockers (tx-scoped vs session-scoped)** — Transaction advisory locks auto-release with the tx (simplest), while session locks survive across statements for longer-lived coordination needing an explicit Releaser.
- **Timeouts use Postgres lock_timeout rather than Go context cancellation** — Cancelling a pgx query mid-lock leaves the connection in an errored tx state (jackc/pgx#2100); the PG-side timeout returns 55P03 while keeping the connection intact.

## Example: Acquire a transaction-scoped business lock

```
import "github.com/openmeterio/openmeter/pkg/framework/lockr"

transaction.RunWithNoValue(ctx, txCreator, func(ctx context.Context) error {
  if err := locker.LockForTXWithScopes(ctx, "subscription", subID); err != nil {
    return err // ErrLockTimeout on 55P03
  }
  return doWork(ctx)
})
```

<!-- archie:ai-end -->
