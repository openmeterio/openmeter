# entdriver

<!-- archie:ai-start -->

> Thin wrapper pairing an entDialectSQL.Driver with an entdb.Client into a single EntPostgresDriver struct — the canonical way to construct and close Ent + Postgres connections in all binaries and tests. Clone() yields isolated clients sharing one connection (used in tests).

## Patterns

**NewEntPostgresDriver from *sql.DB** — Always construct via NewEntPostgresDriver(db *sql.DB) — never instantiate EntPostgresDriver directly — so entDialectSQL.OpenDB(dialect.Postgres, db) wrapping is applied consistently. (`driver := entdriver.NewEntPostgresDriver(sqlDB)`)
**Clone for isolated clients sharing one connection** — Clone() creates a second EntPostgresDriver sharing the underlying *sql.DB but with its own entdb.Client; used in tests to give each test an isolated client. (`isolated := mainDriver.Clone()`)
**Symmetric Close: client first, driver second** — Close() closes the entdb.Client before the entDialectSQL.Driver. Reversing or skipping the client close leaks Ent connection state. (`d.client.Close(); d.driver.Close()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Sole file — defines EntPostgresDriver and NewEntPostgresDriver. Imported by app/common for wiring and openmeter/testutils for test DB setup. | Client() returns the raw *entdb.Client; never call Close() on it separately after handing it to Wire — EntPostgresDriver.Close() owns the lifecycle. |

## Anti-Patterns

- Constructing entdb.NewClient directly in domain or app code instead of going through EntPostgresDriver
- Calling driver.Client().Close() independently — EntPostgresDriver.Close() already does this; double-closing errors
- Passing *entdb.Client across binary boundaries — the client is not serialisable; use Kafka or HTTP

## Decisions

- **Bundle driver and client into one struct** — Ent requires both to be closed in the right order; bundling prevents callers from forgetting one half.

<!-- archie:ai-end -->
