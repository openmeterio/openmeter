# testutils

<!-- archie:ai-start -->

> Self-contained test environment for taxcode integration tests. Wires a real Postgres DB, Ent client, adapter, and service from primitives without importing app/common, preventing import cycles.

## Patterns

**Full stack from primitives without app/common** — NewTestEnv constructs testutils.TestDB → entdb.Client → taxcodeadapter → taxcodeservice.New directly, avoiding the Wire DI graph and any circular import. (`adapter, err := taxcodeadapter.New(taxcodeadapter.Config{Client: client, Logger: logger}); svc, err := taxcodeservice.New(taxcodeservice.Config{Adapter: adapter, Logger: logger})`)
**sync.Once for safe Close** — TestEnv.Close uses sync.Once to prevent double-close when both t.Cleanup and explicit calls race. (`e.close.Do(func() { e.Client.Close(); e.db.EntDriver.Close(); e.db.PGDriver.Close() })`)
**DBSchemaMigrate called explicitly per test** — Tests call env.DBSchemaMigrate(t) before first DB access; this runs client.Schema.Create(t.Context()) on the isolated pgtestdb database. (`env.DBSchemaMigrate(t)`)
**Close order: EntClient then EntDriver then PGDriver** — Close must close the Ent client first, then EntDriver, then PGDriver in that order to avoid leaking the pgtestdb template database. (`e.Client.Close(); e.db.EntDriver.Close(); e.db.PGDriver.Close()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | Sole file. Exports NewTestEnv and TestEnv with Logger, Service, and Client fields for use in _test packages. | Both EntDriver and PGDriver must be closed in Close(); omitting PGDriver leaks the pgtestdb template database. |

## Anti-Patterns

- Importing app/common in testutils — creates import cycles and couples test setup to the full DI graph.
- Calling env.Close before env.DBSchemaMigrate — the schema won't exist for the test.
- Using context.Background() instead of t.Context() in DBSchemaMigrate or service calls.
- Skipping t.Cleanup registration for Close — leaks DB connections if tests fail mid-setup.

## Decisions

- **Build adapter and service directly from constructors rather than via Wire.** — Keeps test helpers independent from app/common so unrelated wiring changes do not create test-only import cycles.

<!-- archie:ai-end -->
