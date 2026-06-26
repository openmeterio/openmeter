# testutils

<!-- archie:ai-start -->

> Shared test fixtures and an in-process TestEnv harness for the product catalog domain. Provides factory functions (NewTestPlan, NewTestAddon, NewTestFeature, NewTestMeters) and a fully wired TestEnv that boots feature/plan/addon/planaddon/taxcode services against a real Postgres instance for integration tests.

## Patterns

**Factory returns Create*Input, not the aggregate** — Test factories return the service's input struct (e.g. plan.CreatePlanInput, addon.CreateAddonInput, feature.CreateFeatureInputs) so tests drive the real service.Create path rather than fabricating domain objects. (`func NewTestPlan(t, namespace, transformers...) plan.CreatePlanInput`)
**Functional-option transformers via generics** — Mutation of fixtures uses TransformerFunc[T any] = func(*testing.T, *T) applied in a loop at the end of the factory. Add new knobs as WithX helpers returning TransformerFunc[productcatalog.Plan], never by adding bool/param args. (`WithPlanPhases(phases...), WithPlanKey(key); applied as `for _, tr := range transformers { tr(t, &input.Plan) }``)
**t.Helper() in every factory** — Every exported factory and transformer calls t.Helper() as its first statement so failures point at the test, not the fixture. (`func NewTestFeature(t *testing.T, namespace string) ... { t.Helper(); ... }`)
**TestEnv wires real adapters+services, not mocks** — NewTestEnv constructs concrete Postgres adapters (productcatalogadapter.NewPostgresFeatureRepo, planadapter.New, addonadapter.New, planaddonadapter.New, taxcodeadapter.New) and real services, plus eventbus.NewMock(t) publisher and meteradapter (mock) — the underlying-constructor approach mandated by AGENTS.md (no app/common wiring). (`featureService := feature.NewFeatureConnector(featureAdapter, meterAdapter, publisher)`)
**Postgres-backed env with idempotent teardown** — DB comes from testutils.InitPostgresDB(t); Close uses sync.Once and closes EntDriver, PGDriver, and Client. DBSchemaMigrate runs Schema.Create on demand. (`e.close.Do(func() { e.db.EntDriver.Close(); e.db.PGDriver.Close(); e.Client.Close() })`)
**ULID for all generated test IDs and namespaces** — NewTestULID is the single ID source; NewTestNamespace is an alias of it. Never hardcode IDs/namespaces — pass a fresh ULID per test for isolation. (`var NewTestNamespace = NewTestULID`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | Defines TestEnv struct and NewTestEnv — the integration harness exposing Feature/Plan/Addon/PlanAddon services + repositories. | Service wiring order matters: featureResolver wraps featureService and feeds both plan and addon services; planaddon depends on both plan and addon services. Uses context.Background() in DBSchemaMigrate/Close (harness lifecycle, predates t.Context()). |
| `plan.go` | NewTestPlan factory plus TransformerFunc/WithPlanPhases/WithPlanKey options and MonthPeriod constant. | Default plan ships a single 'free' phase with a FlatFeeRateCard at amount 0, InArrears term, Stripe tax code txcd_10000000, ProRating enabled (ProratePrices), CreditThenInvoiceSettlementMode. Override via transformers, not by editing the literal. |
| `addon.go` | NewTestAddon — builds addon.CreateAddonInput with AddonInstanceTypeSingle, USD currency, variadic RateCards. | Key/Name are fixed 'test-addon'; for multiple addons in one namespace you must vary them yourself. |
| `feature.go` | NewTestFeature (bare feature) and NewTestFeatureFromMeter (derives key/name/groupby from a meter.Meter). | NewTestFeatureFromMeter uses feature.ConvertMapStringToMeterGroupByFilters and lo.ToPtr(meter.ID) — keep in sync with feature input shape. |
| `meters.go` | NewTestMeters — three canonical meters (count/api_requests, sum/tokens, sum/workload duration). | Each meter gets a fresh NewTestULID ID; ValueProperty is set only on sum meters. |
| `namespace.go` | NewTestULID and the NewTestNamespace alias. | Uses ulid.MustNew with crypto/rand — generation can theoretically fail under entropy starvation but is treated as test-only. |

## Anti-Patterns

- Importing app/common or the DI/wiring layer to build TestEnv — must construct adapters/services from their package constructors to avoid import cycles.
- Adding boolean/positional parameters to factories instead of new WithX transformer options.
- Returning a fully-built productcatalog aggregate from a factory instead of the service Create*Input.
- Hardcoding IDs or namespaces instead of using NewTestULID/NewTestNamespace, breaking per-test isolation.
- Swapping concrete Postgres adapters for mocks in NewTestEnv — these are integration fixtures expecting a real DB.

## Decisions

- **TestEnv composes real Postgres adapters + real services with only the meter adapter and event publisher mocked.** — Catalog tests exercise the genuine persistence and cross-service (plan↔addon↔planaddon↔feature) paths; mocking would hide adapter and SQL behavior.
- **Generic TransformerFunc[T] option pattern over per-field setters.** — Keeps factories a single source of sane defaults while letting each test override only what it cares about, without combinatorial parameter lists.

## Example: Customize a test plan with extra phases in an integration test

```
env := testutils.NewTestEnv(t)
defer env.Close(t)
env.DBSchemaMigrate(t)
ns := testutils.NewTestNamespace(t)
input := testutils.NewTestPlan(t, ns,
  testutils.WithPlanKey("pro"),
  testutils.WithPlanPhases(phase1, phase2),
)
p, err := env.Plan.CreatePlan(t.Context(), input)
require.NoError(t, err)
```

<!-- archie:ai-end -->
