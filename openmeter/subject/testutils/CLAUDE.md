# testutils

<!-- archie:ai-start -->

> Test environment factory for the subject domain — wires subject, customer, entitlement, and feature services directly from package constructors (no app/common) so test suites have a fully functional dependency graph without import cycles.

## Patterns

**NewTestEnv constructs the full graph from package constructors** — Each service is built by calling its own adapter/service constructor (subjectadapter.New, subjectservice.New) rather than importing app/common. (`subjectAdapter, _ := subjectadapter.New(client); subjectService, _ := subjectservice.New(subjectAdapter)`)
**eventbus.NewMock as event publisher** — Tests use eventbus.NewMock(t) instead of a real Kafka publisher; all services accepting a Publisher receive this mock. (`publisher := eventbus.NewMock(t)`)
**testutils.InitPostgresDB for real DB access** — Tests spin up a real Postgres via openmeter/testutils.InitPostgresDB(t). Call env.DBSchemaMigrate(t) before the first DB operation. (`db := testutils.InitPostgresDB(t); client := db.EntDriver.Client()`)
**sync.Once in Close to prevent double-close panics** — TestEnv.Close wraps driver cleanup in sync.Once. Always call t.Cleanup(func() { env.Close(t) }) at test start. (`e.close.Do(func() { e.db.EntDriver.Close(); e.db.PGDriver.Close() })`)
**NewTestULID / NewTestNamespace for collision-free identifiers** — Generate collision-free namespace and ID strings using NewTestULID(t); NewTestNamespace is an alias. (`ns := testutils.NewTestNamespace(t)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | TestEnv struct and NewTestEnv factory; exports SubjectService, CustomerService, EntitlementAdapter, FeatureService, Client. | meteradapter.New(nil) creates a mock meter adapter with no backing store — do not use for tests needing real meter queries. DBSchemaMigrate before any DB writes. Each test should create its own TestEnv to avoid cross-test state. |

## Anti-Patterns

- Importing app/common inside testutils — creates circular imports and couples tests to binary wiring
- Using context.Background() instead of t.Context() in test helper functions
- Skipping env.Close(t) — leaves DB connections open across test runs
- Sharing a single TestEnv across parallel tests without isolation

## Decisions

- **Wire all dependencies from package constructors rather than app/common DI** — Keeps testutils independent of the application wiring layer; unrelated changes to app/common cannot break domain tests and import cycles are impossible.

## Example: Standard test setup using TestEnv

```
func TestSubjectCreate(t *testing.T) {
	env := testutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	ns := testutils.NewTestNamespace(t)
	sub, err := env.SubjectService.Create(t.Context(), subject.CreateInput{Namespace: ns, Key: "test-subject"})
	require.NoError(t, err)
	require.Equal(t, "test-subject", sub.Key)
}
```

<!-- archie:ai-end -->
