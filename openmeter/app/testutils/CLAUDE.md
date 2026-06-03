# testutils

<!-- archie:ai-start -->

> Fully wired app domain test environment backed by a real PostgreSQL database — provides an Env struct with AppService and Adapter pre-wired, helpers for installing/fetching sandbox apps, and a minimal no-op sandbox factory for tests needing app lifecycle without billing logic.

## Patterns

**NewTestEnv wires the full domain stack against a real DB** — NewTestEnv calls testutils.InitPostgresDB, constructs appadapter and appservice, optionally registers a Sandbox factory, and returns Env. Each test gets an isolated DB via pgtestdb. (`env := testutils.NewTestEnv(t, NewEnvConfig{RegisterSandboxFactory: true}); defer env.Close(t)`)
**Conditional Sandbox factory registration** — With BillingService, the full appsandbox.Factory registers (billing logic). With RegisterSandboxFactory but no BillingService, a minimalSandboxFactory (no-op) registers — for CRUD-only tests. (`NewEnvConfig{BillingService: billingSvc} vs NewEnvConfig{RegisterSandboxFactory: true}`)
**DBSchemaMigrate for schema setup** — Tests needing Ent schema must call env.DBSchemaMigrate(t), which runs Schema.Create on the test DB client. (`env.DBSchemaMigrate(t)`)
**sync.Once in Close for safe cleanup** — Env.Close wraps teardown in sync.Once so it is safe from multiple defer sites and t.Cleanup; closes both the Ent and PG drivers. (`defer env.Close(t)`)
**InstallSandboxApp and MustGetApp helpers** — Package-level helpers that assert success and fail the test on error, removing boilerplate. (`installedApp := testutils.InstallSandboxApp(t, env, namespace)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | Env struct, NewTestEnv, DBSchemaMigrate, Close, minimalSandboxFactory, minimalSandboxApp, InstallSandboxApp, MustGetApp. | minimalSandboxFactory.NewApp returns a minimalSandboxApp implementing app.App but not billing.InvoicingApp — tests needing invoice flows must pass a real billing.Service. Do not import app/common here. |

## Anti-Patterns

- Importing app/common from this testutils package — build dependencies must come from underlying constructors to avoid import cycles
- Calling env.DBSchemaMigrate without an initialised DB — the db field must be set (InitPostgresDB does this in NewTestEnv)
- Using minimalSandboxFactory in tests that exercise billing invoice flows — use NewEnvConfig{BillingService: ...}
- Sharing a single Env across multiple test functions — each test should call NewTestEnv for an isolated database

## Decisions

- **minimalSandboxFactory registered separately from full appsandbox.Factory** — Most app-domain tests (CRUD, hooks, lifecycle) do not exercise billing invoice paths; the minimal factory avoids requiring a fully wired billing service, reducing setup complexity.

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
