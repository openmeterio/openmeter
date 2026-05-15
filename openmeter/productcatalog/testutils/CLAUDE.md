# testutils

<!-- archie:ai-start -->

> Test helpers for productcatalog integration tests: TestEnv wires all productcatalog services (feature, plan, planaddon, addon, taxcode) against a real Postgres DB using raw adapter constructors, and factory helpers (NewTestFeature, NewTestPlan, NewTestAddon, NewTestMeters) produce Create*Input fixtures. Primary constraint: never import app/common — wire from raw constructors to avoid import cycles.

## Patterns

**TestEnv with DBSchemaMigrate + Close lifecycle** — NewTestEnv(t) creates the DB and wires all services; always call env.DBSchemaMigrate(t) before any write; env.Close(t) uses sync.Once so it is safe to defer multiple times. (`env := testutils.NewTestEnv(t); env.DBSchemaMigrate(t); defer env.Close(t)`)
**eventbus.NewMock for event assertions** — TestEnv injects eventbus.NewMock(t) as the Watermill publisher so tests can assert events were published without a real Kafka broker. (`publisher := eventbus.NewMock(t)`)
**Factory helpers return input types, not domain entities** — NewTestFeature, NewTestPlan, NewTestAddon, NewTestFeatureFromMeter return Create*Input structs; the caller must call the service to actually persist the entity. (`input := testutils.NewTestFeature(t, ns); feat, err := env.Feature.CreateFeature(ctx, input)`)
**meteradapter.TestAdapter as meter service** — TestEnv uses meteradapter.TestAdapter (SetDBClient(client)) instead of real meter.Service; tests must add meters to the DB client directly to satisfy FK constraints. (`meterAdapter, _ := meteradapter.New(nil); _ = meterAdapter.SetDBClient(client)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | TestEnv struct and NewTestEnv factory wiring all productcatalog services; DBSchemaMigrate and Close lifecycle. | env.Close uses sync.Once — safe to defer multiple times. Do not close Client or db externally. |
| `plan.go` | NewTestPlan with BillingCadence P1M, USD currency, ProRating enabled, CreditThenInvoice settlement. | Default plan has no phases — pass phases as variadic args if rate cards are needed. |
| `feature.go` | NewTestFeature (no meter) and NewTestFeatureFromMeter (meter-backed with eq filters from meter.GroupBy). | NewTestFeatureFromMeter uses ConvertMapStringToMeterGroupByFilters — only creates eq filters, not advanced typed filters. |
| `meters.go` | NewTestMeters returns three canonical meters: api_requests_total (count), tokens_total (sum), workload_runtime_duration_seconds (sum). | IDs generated with NewTestULID(t) — unique per test run; do not hardcode IDs when using these meters. |

## Anti-Patterns

- Importing app/common — use raw adapter constructors as done in NewTestEnv to avoid import cycles.
- Calling env services before DBSchemaMigrate — Ent schema must be applied before any write.
- Sharing a TestEnv across parallel tests without synchronisation — each parallel test should create its own TestEnv.
- Assuming factory helpers persist entities — they return input structs that must be passed to the corresponding service.

## Decisions

- **Wire services from raw constructors (not app/common Wire sets)** — Importing app/common from domain testutils creates import cycles and couples tests to unrelated wiring changes in other domains.

## Example: Set up a productcatalog integration test with a meter-backed feature

```
env := testutils.NewTestEnv(t)
env.DBSchemaMigrate(t)
defer env.Close(t)

meters := testutils.NewTestMeters(t, ns)
// insert meters via env.Client.Meter.Create()...
featureInput := testutils.NewTestFeatureFromMeter(t, ns, meters.ApiRequestsTotal)
feat, err := env.Feature.CreateFeature(t.Context(), featureInput)
require.NoError(t, err)
```

<!-- archie:ai-end -->
