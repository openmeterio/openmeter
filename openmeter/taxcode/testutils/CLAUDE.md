# testutils

<!-- archie:ai-start -->

> Test harness for the taxcode domain: spins up a real Postgres-backed Ent adapter + service and provides fixture helpers (CreateTaxCode, SetupNamespaceDefaults) for taxcode service/adapter tests.

## Patterns

**TestEnv built from concrete constructors** — NewTestEnv wires testutils.InitPostgresDB -> taxcodeadapter.New -> taxcodeservice.New directly, not via app/common DI, to avoid test import cycles. (`adapter, _ := taxcodeadapter.New(taxcodeadapter.Config{Client: client, Logger: logger}); svc, _ := taxcodeservice.New(taxcodeservice.Config{Adapter: adapter, Logger: logger})`)
**Idempotent close via sync.Once** — Close() runs once and shuts down ent client, EntDriver, and PGDriver; tests register t.Cleanup(env.Close). (`e.close.Do(func() { e.Client.Close(); e.db.EntDriver.Close(); e.db.PGDriver.Close() })`)
**Explicit schema migration step** — Tests call env.DBSchemaMigrate(t) which runs Client().Schema.Create against the test DB before exercising the service. (`err := e.db.EntDriver.Client().Schema.Create(t.Context())`)
**Fixture helpers with generated defaults** — CreateTaxCode fills Namespace/Key/Name from testutils.NameGenerator when empty and lets an optional CreateTaxCodeInput override fields; SetupNamespaceDefaults seeds two tax codes and upserts org defaults. (`input.Namespace = namespace; if input.Key == "" { input.Key = generated.Key }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | TestEnv struct, NewTestEnv, DBSchemaMigrate, Close, CreateTaxCode, SetupNamespaceDefaults | NewTestEnv already registers a t.Cleanup(Close); the second t.Cleanup(env.Close) in tests is safe only because Close is sync.Once-guarded. Discard logger via testutils.NewDiscardLogger keeps test output clean. |

## Anti-Patterns

- Importing app/common wiring to build the service, which creates test-only import cycles — build from taxcodeadapter.New / taxcodeservice.New instead
- Sharing a single namespace across subtests that mutate org defaults; helpers use freshly generated namespaces per scenario
- Forgetting env.DBSchemaMigrate(t) before service calls, so the taxcode tables don't exist

## Decisions

- **Harness constructs adapter+service from underlying constructors rather than DI** — Keeps taxcode test dependencies independent from app/common so unrelated wiring additions cannot introduce import cycles into this package.

## Example: Standard taxcode test setup

```
env := taxcodetestutils.NewTestEnv(t)
t.Cleanup(func() { env.Close(t) })
env.DBSchemaMigrate(t)
ns := testutils.NameGenerator.Generate().Key
env.SetupNamespaceDefaults(t.Context(), t, ns)
```

<!-- archie:ai-end -->
