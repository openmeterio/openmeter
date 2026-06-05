# pglockx

<!-- archie:ai-start -->

> Thin wrapper around cirello.io/pglock that constructs a Postgres-backed distributed lock client over the `distributed_locks` table. Its primary constraint: the lock client is created via pglock.UnsafeNew and depends on validated lease/heartbeat timing.

## Patterns

**Config.Validate before construction** — New() calls config.Validate() first and refuses to build a client on invalid config, wrapping the error. (`if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid lock configuration: %w", err) }`)
**Lease/heartbeat invariant** — Validate enforces LeaseTime must be at least twice HeartbeatInterval, and Owner must be non-empty; errors collected via errors.Join. (`if c.LeaseTime/2 < c.HeartbeatInterval { errs = append(errs, errors.New(...)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pglock.go` | Defines Config (LeaseTime, HeartbeatInterval, Owner), Validate(), and New(db, config) returning *pglock.Client over the hardcoded `distributed_locks` table. | Uses pglock.UnsafeNew (not New) — table is fixed via WithCustomTable(lockTable). DefaultHeartbeatInterval=3s, DefaultLeaseTime=1m; preserve the lease>=2*heartbeat relationship if changing defaults. |

## Anti-Patterns

- Calling pglock.New/UnsafeNew directly instead of going through New() and skipping config validation.
- Setting HeartbeatInterval too close to LeaseTime (violates the 2x lease invariant).

## Decisions

- **Use Postgres-based pglock with a custom `distributed_locks` table.** — Reuses the existing Postgres dependency for distributed leader/worker locking instead of adding a separate coordination service.

<!-- archie:ai-end -->
