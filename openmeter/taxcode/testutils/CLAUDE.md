# testutils

<!-- archie:ai-start -->

> Self-contained test environment for taxcode integration tests: wires a real Postgres DB, Ent client, adapter, and service without importing app/common.

## Patterns

**NewTestEnv constructs the full stack from primitives** — TestEnv builds testutils.TestDB → entdb.Client → taxcodeadapter → taxcodeservice.New without touching app/common Wire providers, avoiding import cycles. (`adapter, err := taxcodeadapter.New(taxcodeadapter.Config{Client: client, Logger: logger}); env.Service = taxcodeservice.New(adapter, logger)`)
**sync.Once for safe Close** — TestEnv.Close uses sync.Once to prevent double-close when both t.Cleanup and explicit calls race. (`e.close.Do(func() { ... })`)
**DBSchemaMigrate called explicitly in each test** — Tests call env.DBSchemaMigrate(t) before first DB access; this runs client.Schema.Create(t.Context()) to ensure the schema is up-to-date for the isolated pgtestdb database. (`env.DBSchemaMigrate(t)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | Sole file; exports NewTestEnv and TestEnv with Logger, Service, Client fields. | Close must close both EntDriver and PGDriver in order; omitting either leaks the pgtestdb template database. |

## Anti-Patterns

- Importing app/common in testutils — creates import cycles and couples test setup to the full DI graph.
- Calling env.Close before DBSchemaMigrate — schema won't exist for the test.
- Using context.Background() instead of t.Context() in DBSchemaMigrate or service calls.

## Decisions

- **Build adapter and service directly from constructors rather than via Wire.** — Keeps test helpers independent from app/common so unrelated wiring changes don't create test-only import cycles.

<!-- archie:ai-end -->
