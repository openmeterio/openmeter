# testutils

<!-- archie:ai-start -->

> Fully wired app domain test environment backed by a real PostgreSQL database — provides Env struct with AppService and Adapter pre-wired, convenience helpers for installing sandbox apps and fetching apps, and a minimal no-op sandbox factory for tests that need app lifecycle without billing logic.

## Patterns

**NewTestEnv wires the full domain stack against a real DB** — NewTestEnv calls testutils.InitPostgresDB, constructs appadapter and appservice, optionally registers the Sandbox factory (full or minimal), and returns Env. Each test gets an isolated DB via pgtestdb. (`env := testutils.NewTestEnv(t, NewEnvConfig{RegisterSandboxFactory: true}); defer env.Close(t)`)
**Conditional Sandbox factory registration** — If BillingService is provided, the full appsandbox.Factory is registered (enables billing logic). If RegisterSandboxFactory is true without BillingService, a minimalSandboxFactory (no-op) is registered. This covers tests that need CRUD operations but not invoice flows. (`NewEnvConfig{BillingService: billingSvc} vs NewEnvConfig{RegisterSandboxFactory: true}`)
**DBSchemaMigrate for schema setup** — Tests that need Ent schema must call env.DBSchemaMigrate(t) explicitly. It runs Schema.Create on the test DB client. (`env.DBSchemaMigrate(t)`)
**sync.Once in Close for safe cleanup** — Env.Close wraps teardown in sync.Once so it is safe to call from multiple defer sites and from t.Cleanup. Closes both the Ent driver and the PG driver. (`defer env.Close(t)`)
**InstallSandboxApp and MustGetApp helpers** — Package-level helpers that assert success and fail the test on error — removing boilerplate from individual tests. (`installedApp := testutils.InstallSandboxApp(t, env, namespace)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | Env struct, NewTestEnv, DBSchemaMigrate, Close, minimalSandboxFactory, minimalSandboxApp, InstallSandboxApp, MustGetApp. | minimalSandboxFactory.NewApp returns a minimalSandboxApp that implements app.App but not billing.InvoicingApp — tests needing invoice flows must pass a real billing.Service in NewEnvConfig. Do not import app/common from here. |

## Anti-Patterns

- Importing app/common from this testutils package — build dependencies must come from underlying package constructors to avoid import cycles.
- Calling env.DBSchemaMigrate without initialising the DB first — the db field must be set (InitPostgresDB already does this in NewTestEnv).
- Using minimalSandboxFactory in tests that exercise billing invoice flows — use NewEnvConfig{BillingService: ...} instead.
- Sharing a single Env across multiple test functions — each test should call NewTestEnv to get an isolated database.

## Decisions

- **minimalSandboxFactory registered separately from full appsandbox.Factory** — Most app-domain tests (CRUD, hooks, lifecycle) do not exercise billing invoice paths; the minimal factory avoids requiring a fully wired billing service for those tests, reducing test setup complexity.

## Example: Setting up an app test environment and installing a sandbox app

```
func TestFoo(t *testing.T) {
	env := apptestutils.NewTestEnv(t, apptestutils.NewEnvConfig{RegisterSandboxFactory: true})
	defer env.Close(t)
	env.DBSchemaMigrate(t)
	installedApp := apptestutils.InstallSandboxApp(t, env, "default")
	// ... test against installedApp ...
}
```

<!-- archie:ai-end -->
