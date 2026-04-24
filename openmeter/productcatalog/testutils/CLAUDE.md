# testutils

<!-- archie:ai-start -->

> Test helpers for productcatalog integration tests: TestEnv wiring all productcatalog services against a real Postgres DB, and factory helpers for Feature, Plan, Addon, Meter, and Namespace fixtures.

## Patterns

**TestEnv via NewTestEnv** — NewTestEnv(t) spins up a real Postgres DB via testutils.InitPostgresDB, wires feature/plan/addon/planaddon services from raw adapters (not app/common), and exposes all services. Always call env.DBSchemaMigrate(t) before use. (`env := testutils.NewTestEnv(t); env.DBSchemaMigrate(t); defer env.Close(t)`)
**eventbus.NewMock for event assertions** — TestEnv uses eventbus.NewMock(t) as the publisher so tests can assert Watermill events were published without needing a real Kafka. (`publisher := eventbus.NewMock(t)`)
**Factory helpers return input types not domain types** — NewTestFeature, NewTestPlan, NewTestAddon, NewTestFeatureFromMeter return Create*Input structs, not the created entities — caller must call the service to persist. (`input := testutils.NewTestFeature(t, ns); feat, err := env.Feature.CreateFeature(ctx, input)`)
**meteradapter.TestAdapter as meter service** — TestEnv uses meteradapter.TestAdapter (SetDBClient(client)) instead of real meter.Service; tests must add meters to the DB client directly. (`meterAdapter, _ := meteradapter.New(nil); _ = meterAdapter.SetDBClient(client)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | TestEnv struct and NewTestEnv factory that wires all productcatalog services; DBSchemaMigrate and Close lifecycle. | env.Close uses sync.Once — safe to defer multiple times. Client and db are both closed; do not close them externally. |
| `plan.go` | NewTestPlan with BillingCadence P1M, USD currency, ProRating enabled, CreditThenInvoice settlement. | Default plan has no phases — pass phases as variadic args if rate cards are needed. |
| `feature.go` | NewTestFeature (no meter) and NewTestFeatureFromMeter (meter-backed). | NewTestFeatureFromMeter uses ConvertMapStringToMeterGroupByFilters on meter.GroupBy — only creates eq filters. |
| `meters.go` | NewTestMeters returns three canonical meters: api_requests_total (count), tokens_total (sum), workload_runtime_duration_seconds (sum). | IDs are generated with NewTestULID(t) — unique per test run, do not hardcode IDs when using these. |

## Anti-Patterns

- Importing app/common in tests — use raw adapter constructors as done in NewTestEnv to avoid import cycles.
- Calling env services before DBSchemaMigrate — Ent schema must be applied before any write.
- Sharing a TestEnv across parallel tests without a mutex — each parallel test should create its own TestEnv.

## Decisions

- **Wire services from raw constructors (not app/common Wire sets)** — Importing app/common from domain testutils creates import cycles and couples tests to unrelated wiring changes.

<!-- archie:ai-end -->
