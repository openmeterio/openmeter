# testutils

<!-- archie:ai-start -->

> Shared test fixtures and dependency builders for the subscription domain: SubscriptionDependencies (a fully wired set of real services from Ent adapters), example entities, comparison helpers, and mocks. Must NOT import app/common (import-cycle boundary).

## Patterns

**NewService builds the full dependency graph from constructors** — NewService(t, dbDeps) wires customer/subject/entitlement/plan/addon/subscription/workflow from their underlying constructors, not app/common Wire providers. (`deps := subscriptiontestutils.NewService(t, dbDeps)`)
**SetupDBDeps provisions a real migrated Postgres** — SetupDBDeps(t) calls testutils.InitPostgresDB, runs migrate.Up(), returns DBDeps; tests must register deps.Cleanup via t.Cleanup. (`dbDeps := subscriptiontestutils.SetupDBDeps(t); t.Cleanup(func() { dbDeps.Cleanup(t) })`)
**Example* vars are canonical test data; clone before use** — ExampleRateCard1–5 etc. are package-level vars; always Clone() before using as plan/subscription input to avoid cross-test mutation. (`b.AddPhase(lo.ToPtr(ISOMonth), ExampleRateCard1.Clone())`)
**CreateTestAddon does create+publish** — deps.AddonService.CreateTestAddon(t, inp) creates and immediately publishes the addon (the required two-step flow). (`add := deps.AddonService.CreateTestAddon(t, addonInp)`)
**MockService / MockWorkflowService with Fn fields** — Wire only the Fn fields for methods under test; leave others nil to panic on unexpected calls. (`svc := &subscriptiontestutils.MockService{GetViewFn: func(...) {...}}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | NewService — canonical wiring entry point; SubscriptionDependencies holds all assembled services. | After SetDBClient, read back meterID via GetMeterByIDOrSlug — pgtestdb template reuse can resolve a different ID. |
| `db.go` | SetupDBDeps with mutex-guarded Postgres provisioning and full migration. | sync.Mutex serialises DB setup; parallel subtests must each call SetupDBDeps. |
| `compare.go` | ValidateSpecAndView deep-compares spec vs view incl. entitlement/feature/phase alignment. | UsagePeriod alignment uses Truncate(time.Minute) — avoid sub-minute precision in fixtures. |
| `helpers.go` | CreatePlanWithAddon, CreateSubscriptionFromPlan, CreateAddonForSubscription scenario builders. | CreateAddonForSubscription encodes a Quantity=0 terminal entry; use CreateMultiInstanceAddonForSubscription for multi-instance. |
| `builder.go` | testPlanbuilder and testSubscriptionSpecBuilder fluent DSLs. | Build() calls SyncAnnotations + Validate — always use Build(), not the spec directly. |
| `mock.go` | MockService and MockWorkflowService. | MockWorkflowService does not implement Restore/AddAddon/ChangeAddonQuantity — calling them panics. |

## Anti-Patterns

- Importing app/common from testutils — causes import cycles.
- Using ExampleRateCard* vars directly without Clone() — shared vars mutate across tests.
- Calling service.CreateAddon without PublishAddon — use CreateTestAddon.
- Constructing SubscriptionDependencies manually instead of via NewService — misses required hook registration.
- Using context.Background() instead of t.Context() in helpers.

## Decisions

- **All services built from package constructors, not app/common Wire providers.** — Prevents import cycles and keeps test wiring independent of production DI changes.
- **ValidateSpecAndView validates entitlement alignment against billingAnchor.** — Entitlement UsagePeriod must anchor to subscription BillingAnchor — a billing correctness invariant tests must enforce.

## Example: Set up a full subscription integration test with plan and addon

```
func TestMyFeature(t *testing.T) {
  dbDeps := subscriptiontestutils.SetupDBDeps(t)
  t.Cleanup(func() { dbDeps.Cleanup(t) })
  deps := subscriptiontestutils.NewService(t, dbDeps)
  plan, addon := subscriptiontestutils.CreatePlanWithAddon(t, deps,
    subscriptiontestutils.GetExamplePlanInput(t),
    subscriptiontestutils.GetExampleAddonInput(t, period))
  subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, plan, clock.Now())
  subscriptiontestutils.ValidateSpecAndView(t, expectedSpec, subView)
}
```

<!-- archie:ai-end -->
