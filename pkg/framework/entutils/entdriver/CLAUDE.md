# entdriver

<!-- archie:ai-start -->

> Wraps an Ent Postgres client + dialect driver pair into a single EntPostgresDriver lifecycle object, providing Driver()/Client() accessors and Clone() for sharing the underlying *sql.DB across multiple Ent clients. The one hand-written runtime glue type tying database/sql to the generated openmeter/ent/db client.

## Patterns

**Driver+client pairing** — EntPostgresDriver holds both *entDialectSQL.Driver and *entdb.Client built from the same connection; construct via NewEntPostgresDriver(db *sql.DB) which opens the dialect driver and wraps it in entdb.NewClient. (`driver := entDialectSQL.OpenDB(dialect.Postgres, db); client := entdb.NewClient(entdb.Driver(driver))`)
**Shared-connection Clone** — Clone() reuses d.driver.DB() (the same *sql.DB) to build a fresh driver+client, enabling multiple Ent clients over one pool/connection (used for transaction sharing). (`driver := entDialectSQL.OpenDB(dialect.Postgres, d.driver.DB())`)
**Ordered Close** — Close() closes the client first, then the dialect driver, returning the first error; both resources are owned by this struct. (`if err := d.client.Close(); err != nil { return err }; return d.driver.Close()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Defines EntPostgresDriver and its New/Clone/Close/Driver/Client methods over openmeter/ent/db | Imports the concrete generated entdb package — this file is the bridge to that codegen output; Clone shares the *sql.DB so closing one driver affects the shared pool |

## Anti-Patterns

- Constructing entdb.Client directly in callers instead of through NewEntPostgresDriver, bypassing paired lifecycle
- Closing the shared *sql.DB via one Clone()'d driver while another clone is still in use
- Reordering Close() so the dialect driver closes before the client

## Decisions

- **Keep both driver and client in one struct with explicit accessors** — Callers (transaction helpers, testutils, lockr) need either the low-level driver or the typed client from a single owned lifecycle
- **Provide Clone() over the underlying *sql.DB** — Enables shared-transaction patterns where multiple Ent clients operate on the same connection

<!-- archie:ai-end -->
