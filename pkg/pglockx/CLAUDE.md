# pglockx

<!-- archie:ai-start -->

> Thin constructor for cirello.io/pglock distributed lock clients backed by a PostgreSQL table (distributed_locks). Enforces the invariant that HeartbeatInterval must be less than half the LeaseTime to prevent lock expiry before renewal.

## Patterns

**Validate config before constructing the client** — Config.Validate() enforces LeaseTime >= 2*HeartbeatInterval and a non-empty Owner. Call Validate() or rely on New() which calls it internally — never pass an unvalidated Config to pglock.UnsafeNew. (`if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid lock configuration: %w", err) }`)
**Use DefaultHeartbeatInterval and DefaultLeaseTime** — DefaultHeartbeatInterval (3s) and DefaultLeaseTime (1m) satisfy the heartbeat < lease/2 invariant. Prefer them when no specific timing is required. (`cfg := pglockx.Config{LeaseTime: pglockx.DefaultLeaseTime, HeartbeatInterval: pglockx.DefaultHeartbeatInterval, Owner: "billing-worker"}`)
**Fixed table name — never override per caller** — The lock table is the hardcoded unexported constant lockTable ("distributed_locks"). All lock clients share this table — do not create per-feature lock tables. (`pglock.WithCustomTable(lockTable) // internal; not configurable by callers`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pglock.go` | Entire package — Config, Validate(), New() wrapping pglock.UnsafeNew with fixed table and configurable lease/heartbeat/owner. | pglock.UnsafeNew is used (not pglock.New) because the safe constructor requires a createTable migration conflicting with Atlas-managed schema. Do not switch to pglock.New without adding distributed_locks to Atlas migrations. |

## Anti-Patterns

- Setting HeartbeatInterval >= LeaseTime/2 — Validate() rejects it; the heartbeat must renew the lease well before expiry.
- Leaving Owner empty — the lock client requires an owner string to identify the holding process.
- Creating per-feature lock tables by overriding WithCustomTable — all locks share the distributed_locks table.
- Using pglock.New instead of pglock.UnsafeNew without adding the distributed_locks table to Atlas migrations.

## Decisions

- **pglock.UnsafeNew over pglock.New.** — pglock.New auto-creates the lock table at startup, conflicting with Atlas-managed migrations. UnsafeNew skips auto-create; the table is created via the Atlas pipeline.
- **Single hardcoded lock table name (distributed_locks).** — A single shared table avoids schema proliferation and keeps the Atlas migration surface minimal; all lock use cases share the table with owner-scoped rows.

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
if err != nil { return fmt.Errorf("pglock client: %w", err) }
```

<!-- archie:ai-end -->
