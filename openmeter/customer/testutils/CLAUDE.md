# testutils

<!-- archie:ai-start -->

> Isolated test environment factory for the customer domain — constructs a fully wired CustomerService + SubjectService against a real pgtestdb PostgreSQL instance without importing app/common, preventing test-only import cycles.

## Patterns

**TestEnv struct with Close via sync.Once** — TestEnv holds all constructed services and the raw DB handles. Close() uses sync.Once to safely close ent driver and pgx pool exactly once even when called multiple times from deferred teardown. (`e.close.Do(func() { e.db.EntDriver.Close(); e.db.PGDriver.Close() })`)
**Build from package constructors, not app/common** — NewTestEnv constructs customeradapter.New, customerservice.New, subjectadapter.New, subjectservice.New directly — no app/common imports, no Wire, no DI container. (`customerAdapter, err := customeradapter.New(customeradapter.Config{Client: client, Logger: logger})`)
**eventbus.NewMock for publisher** — Uses eventbus.NewMock(t) instead of a real Kafka publisher so tests do not require a running Kafka broker. (`publisher := eventbus.NewMock(t)`)
**t.Context() for test-scoped context** — Tests calling NewTestEnv should use t.Context() (not context.Background()) when passing context to service calls so cancellation is tied to the test lifecycle. (`e.CustomerService.CreateCustomer(t.Context(), input)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | TestEnv struct, NewTestEnv factory, DBSchemaMigrate helper, Close teardown. Single file for the entire test environment. | DBSchemaMigrate must be called before tests that require the schema; skipping it causes 'table does not exist' errors. Close must be deferred in every test using NewTestEnv. |

## Anti-Patterns

- Importing app/common from testutils — creates import cycles and makes tests dependent on the full DI graph.
- Using context.Background() in tests instead of t.Context() — misses test-lifecycle cancellation.
- Calling e.Close() without defer — leaks DB connections if the test panics.
- Adding business logic or assertions to testutils — keep it as a pure environment factory.

## Decisions

- **testutils constructs adapters and services directly instead of using app/common Wire providers.** — Avoids import cycles between domain test helpers and the application wiring layer; lets domain tests compile without pulling in unrelated domain providers.

<!-- archie:ai-end -->
