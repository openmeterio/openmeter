# testutils

<!-- archie:ai-start -->

> Shared test fixtures and harness for the subscription domain. Provides SetupDBDeps (migrated Postgres), NewService (a fully wired SubscriptionDependencies graph), example plans/ratecards/customers/addons, spec/view comparators, and mock service implementations.

## Patterns

**Wire real services from underlying constructors** — NewService builds the whole dependency graph from concrete adapters/services (repos, customer, entitlement registry, plan, addon, tax code, workflow) — not from app/common — to avoid test-only import cycles. (`svc, err := service.New(service.ServiceConfig{ SubscriptionRepo: subRepo, ... })`)
**DB harness: SetupDBDeps + DBDeps.Cleanup** — SetupDBDeps initializes a Postgres test DB, runs migrate.OMMigrationsConfig Up under a global mutex, and returns DBDeps{DBClient, EntDriver, PGDriver}; pair with defer dbDeps.Cleanup(t). (`dbDeps := subscriptiontestutils.SetupDBDeps(t); defer dbDeps.Cleanup(t)`)
**Embed-and-extend test wrappers** — Helpers embed the real interface and add t.Helper Create*-style methods (testCustomerRepo, testFeatureConnector, testAddonService, testSubscriptionRepo) that t.Fatalf on error. (`type testCustomerRepo struct { customer.Adapter; subjectService subject.Service }`)
**Exported example fixtures as package vars** — Reusable constants/vars: ExampleNamespace, ExampleFeatureKey(1-3), ExampleRateCard1..5, ExampleAddonRateCard1..6, ExampleCreateCustomerInput, ISOMonth, GetExamplePlanInput. (`plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))`)
**Spec/view comparators** — ValidateSpecAndView asserts a created view matches its source spec (incl. entitlement alignment to billing anchor and tax code backfill); SpecsEqual / SubscriptionAddonsEqual compare specs and addons. (`subscriptiontestutils.ValidateSpecAndView(t, spec, found)`)
**Builders for plans and specs** — BuildTestPlanInput / BuildTestSubscriptionSpec return fluent builders (AddPhase, SetMeta, Build) generating test_phase_N keys; Build validates and syncs annotations. (`BuildTestPlanInput(t).AddPhase(lo.ToPtr(datetime.MustParseDuration(t, "P1M")), ExampleRateCard1.Clone())`)
**Mocks implement the domain interfaces by Fn fields** — MockService and MockWorkflowService satisfy subscription.Service / workflow.Service by delegating to assignable *Fn function fields. (`var _ subscription.Service = &MockService{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | NewService: builds SubscriptionDependencies (the canonical integration-test fixture) | Uses a mock streaming connector and noop tracer; after meterAdapter.SetDBClient it re-reads the resolved meter ID (shared template DB may reassign it). MultiSubscriptionEnabledFF is false by default. Wires annotation cleanup hook and the workflow service. |
| `db.go` | SetupDBDeps / DBDeps / Cleanup | SetupDBDeps serializes via a package-level sync.Mutex and runs full migrations each call; always defer Cleanup or DB connections leak. |
| `compare.go` | ValidateSpecAndView, SpecsEqual, SubscriptionAddonsEqual and time helpers | ValidateSpecAndView checks entitlement UsagePeriod alignment to the truncated billing anchor and metered MeasureUsageFrom == phase start; tax-code expectations branch on TaxConfig.TaxCodeID vs Stripe.Code backfill. |
| `ratecard.go / addon.go / plan.go / feature.go` | Example RateCards, Addons, plan input, and feature fixtures | GetExamplePlanInput builds a fixed 3-phase plan (P1M, P2M, open-ended). Clone() rate cards before mutating shared vars to avoid cross-test contamination. |
| `customer.go / repository.go / builder.go` | Customer service/adapter helpers, repo wrappers, and plan/spec builders | CreateExampleCustomer creates the subjects first (UsageAttribution.SubjectKeys) then the customer; builders use clock.Now() so freeze the clock for determinism. |
| `mock.go` | MockService / MockWorkflowService with *Fn delegation | Unset *Fn fields panic with a nil-call; set every Fn the code under test will invoke. MarshalJSON/UnmarshalJSON on TestPatch panic by design. |
| `billing.go` | NoopCustomerOverrideService satisfying billing.CustomerOverrideService | Returns zero values; use only when billing behavior is irrelevant to the test. |

## Anti-Patterns

- Importing app/common wiring here — build dependencies from underlying constructors to prevent test-only import cycles.
- Mutating a shared Example* RateCard/var without Clone(), leaking state across tests.
- Skipping defer dbDeps.Cleanup(t), leaking Postgres/ent/pg drivers.
- Leaving a MockService *Fn nil for a method the test exercises (panics on nil call).
- Assuming the meter ID stays constant across SetDBClient — read back the resolved ID.

## Decisions

- **NewService assembles the real service graph instead of using the app DI layer** — Keeps subscription test helpers independent from app/common so unrelated wiring changes don't create import cycles, while still exercising production code paths.
- **SetupDBDeps runs full migrations per call under a global mutex** — Guarantees a real, schema-correct Postgres for each suite and serializes setup to avoid concurrent migration races against a shared instance.

## Example: Standard DB-backed subscription test setup

```
dbDeps := subscriptiontestutils.SetupDBDeps(t)
defer dbDeps.Cleanup(t)

deps := subscriptiontestutils.NewService(t, dbDeps)
service := deps.SubscriptionService

cust := deps.CustomerAdapter.CreateExampleCustomer(t)
_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
	CustomerId: cust.ID, Currency: "USD", ActiveFrom: clock.Now(), BillingAnchor: clock.Now(), Name: "Test",
})
require.NoError(t, err)
sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec)
```

<!-- archie:ai-end -->
