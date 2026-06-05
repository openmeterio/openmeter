# testutils

<!-- archie:ai-start -->

> Shared test harness for the customer domain. Provides a TestEnv that wires a real Postgres-backed customer + subject service stack for integration tests, plus ULID/namespace helpers.

## Patterns

**TestEnv built from concrete constructors** — NewTestEnv wires the stack directly: testutils.InitPostgresDB, eventbus.NewMock, meteradapter (mockadapter), subjectadapter/subjectservice, customeradapter/customerservice — never via app/common DI, avoiding test-only import cycles. (`customerAdapter, _ := customeradapter.New(customeradapter.Config{Client: client, Logger: logger}); customerService, _ := customerservice.New(customerservice.Config{Adapter: customerAdapter, Publisher: publisher})`)
**Lazy schema migration + once-guarded close** — DBSchemaMigrate runs Schema.Create(t.Context()) on demand; Close uses sync.Once to close ent/pg drivers and the client exactly once. (`e.close.Do(func(){ e.db.EntDriver.Close(); e.db.PGDriver.Close(); e.Client.Close() })`)
**Discard logger and noop tracer in tests** — Uses testutils.NewDiscardLogger(t) and noop.NewTracerProvider().Tracer for observability deps so tests stay silent and dependency-free. (`logger := testutils.NewDiscardLogger(t); tracer := noop.NewTracerProvider().Tracer("test_env")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | TestEnv struct + NewTestEnv/Close/DBSchemaMigrate, and NewTestULID (aliased as NewTestNamespace). | Meter uses meter/mockadapter.New(nil); event publisher is eventbus.NewMock(t); caller must invoke DBSchemaMigrate before using the DB. |

## Anti-Patterns

- Importing app/common wiring here — build services from underlying constructors to avoid test-only import cycles
- Sharing a TestEnv across tests instead of one per test (it owns a t-scoped Postgres DB)
- Forgetting DBSchemaMigrate before exercising the adapter/service

## Decisions

- **Construct the customer/subject stack from package constructors rather than DI wiring** — Keeps testutils independent of app/common so unrelated wiring additions can't introduce import cycles in domain tests.

## Example: Wiring the customer service stack for tests

```
customerAdapter, err := customeradapter.New(customeradapter.Config{Client: client, Logger: logger})
require.NoErrorf(t, err, "initializing customer adapter must not fail")
customerService, err := customerservice.New(customerservice.Config{Adapter: customerAdapter, Publisher: publisher})
require.NoErrorf(t, err, "initializing customer service must not fail")
```

<!-- archie:ai-end -->
