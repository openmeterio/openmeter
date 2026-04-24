# entdriver

<!-- archie:ai-start -->

> Thin wrapper that pairs an `entDialectSQL.Driver` with an `entdb.Client` into a single `EntPostgresDriver` struct, providing the canonical way to construct and close Ent + Postgres connections in all binaries and tests.

## Patterns

**NewEntPostgresDriver from *sql.DB** — Always construct via `NewEntPostgresDriver(db *sql.DB)` — never instantiate `EntPostgresDriver` directly. This ensures the `entDialectSQL.OpenDB(dialect.Postgres, db)` wrapping is applied consistently. (`driver := entdriver.NewEntPostgresDriver(sqlDB)`)
**Clone for isolated clients sharing one connection** — Use `Clone()` to create a second `EntPostgresDriver` that shares the underlying `*sql.DB` but has its own `entdb.Client`. Used in tests to give each test an isolated client. (`isolated := mainDriver.Clone()`)
**Symmetric Close: client first, driver second** — `Close()` closes the `entdb.Client` before the `entDialectSQL.Driver`. Reversing this order or skipping client close leaks Ent connection state. (`d.client.Close(); d.driver.Close()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Sole file in this package — defines `EntPostgresDriver` and `NewEntPostgresDriver`. Imported by `app/common/database.go` for wiring and by `openmeter/testutils` for test DB setup. | `Client()` returns the raw `*entdb.Client`; callers should never call `Close()` on it separately after handing it to Wire — `EntPostgresDriver.Close()` owns the lifecycle. |

## Anti-Patterns

- Constructing `entdb.NewClient` directly in domain or app code instead of going through `EntPostgresDriver` — bypasses the shared lifecycle management.
- Calling `driver.Client().Close()` independently — `EntPostgresDriver.Close()` already does this; double-closing causes errors.
- Passing `*entdb.Client` across binary boundaries — the client is not serialisable; use Kafka or HTTP for cross-binary communication.

## Decisions

- **Bundle driver and client into one struct** — Ent requires both to be closed in the right order; bundling prevents callers from forgetting one half.

<!-- archie:ai-end -->
