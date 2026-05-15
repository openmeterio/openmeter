# pglockx

<!-- archie:ai-start -->

> Thin constructor for cirello.io/pglock distributed lock clients backed by a PostgreSQL table (distributed_locks). Enforces the invariant that HeartbeatInterval must be less than half the LeaseTime to prevent lock expiry before renewal.

## Patterns

**Validate config before constructing the client** — Config.Validate() enforces two rules: LeaseTime >= 2*HeartbeatInterval and Owner must be non-empty. Call Validate() or rely on New() which calls it internally — never pass an unvalidated Config to pglock.UnsafeNew. (`if err := config.Validate(); err != nil {
    return nil, fmt.Errorf("invalid lock configuration: %w", err)
}`)
**Use DefaultHeartbeatInterval and DefaultLeaseTime as safe starting values** — The package exports DefaultHeartbeatInterval (3s) and DefaultLeaseTime (1m) which satisfy the heartbeat < lease/2 invariant. Prefer these defaults when no specific timing is required. (`cfg := pglockx.Config{
    LeaseTime:         pglockx.DefaultLeaseTime,
    HeartbeatInterval: pglockx.DefaultHeartbeatInterval,
    Owner:             "billing-worker",
}`)
**Fixed table name — never override per caller** — The lock table is hardcoded as the unexported constant lockTable ("distributed_locks"). All lock clients in the codebase share this table — do not create per-feature lock tables. (`pglock.WithCustomTable(lockTable) // internal; not configurable by callers`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pglock.go` | Entire package — Config struct, Validate(), New() constructor wrapping pglock.UnsafeNew with fixed table and configurable lease/heartbeat/owner. | pglock.UnsafeNew is used (not pglock.New) because the safe constructor requires a createTable migration that conflicts with Atlas-managed schema. Do not switch to pglock.New without adding the distributed_locks table to Atlas migrations. |

## Anti-Patterns

- Setting HeartbeatInterval >= LeaseTime/2 — Validate() rejects this but the invariant must be understood: the heartbeat must renew the lease well before expiry
- Leaving Owner empty — the lock client requires an owner string to identify which process holds the lock
- Creating per-feature lock tables by overriding WithCustomTable — all locks share the distributed_locks table
- Using pglock.New instead of pglock.UnsafeNew without adding the distributed_locks table to Atlas migrations

## Decisions

- **pglock.UnsafeNew over pglock.New** — pglock.New attempts to create the lock table at startup, which conflicts with Atlas-managed schema migrations. UnsafeNew skips the auto-create; the table is created via the Atlas migration pipeline.
- **Single hardcoded lock table name (distributed_locks)** — A single shared table avoids schema proliferation and keeps the Atlas migration surface minimal. All distributed lock use cases in the codebase share the same table with owner-scoped rows.

## Example: Construct a pglock client for a billing worker process

```
import (
    "database/sql"
    "github.com/openmeterio/openmeter/pkg/pglockx"
)

client, err := pglockx.New(db, pglockx.Config{
    LeaseTime:         pglockx.DefaultLeaseTime,
    HeartbeatInterval: pglockx.DefaultHeartbeatInterval,
    Owner:             "billing-worker-instance-1",
})
if err != nil {
    return fmt.Errorf("pglock client: %w", err)
}
```

<!-- archie:ai-end -->
