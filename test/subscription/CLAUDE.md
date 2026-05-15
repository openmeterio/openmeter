# subscription

<!-- archie:ai-start -->

> Integration test suite for subscription lifecycle scenarios — verifying the full billing+sync stack (plan creation, subscription create/edit/cancel, billing sync, invoice line generation) against a real PostgreSQL database via pgtestdb. Tests are scenario-based and exercise cross-domain flows that unit tests cannot cover.

## Patterns

**Single shared setup() function via testDeps** — All tests call setup(t, setupConfig{}) which wires every dependency from scratch using direct constructors (no app/common imports). The returned testDeps struct embeds subscriptiontestutils.SubscriptionDependencies and adds billing, sync, and app services. Always defer tDeps.cleanup(t). (`tDeps := setup(t, setupConfig{}); defer tDeps.cleanup(t)`)
**clock.SetTime for deterministic time control** — Tests set a fixed start time via testutils.GetRFC3339Time then advance with clock.SetTime(currentTime.Add(...)). All services use pkg/clock so this controls all time-dependent billing logic. (`currentTime := testutils.GetRFC3339Time(t, "2025-01-20T13:11:07Z"); clock.SetTime(currentTime)`)
**Numbered step comments for scenario readability** — Test bodies use numbered comments (// 1st, // 2nd, etc.) documenting the causal sequence: feature → plan → publish → customer → subscription → mutations → assertions. (`// 1st, let's create the features ... // 2nd, let's create the plan ... // 4th, let's create the subscription`)
**Plan published before subscription creation** — Every scenario calls PlanService.PublishPlan with EffectiveFrom before creating a subscription via pcSubscriptionService.Create. An unpublished plan causes subscription creation to fail. (`p, err = tDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{NamespacedID: p.NamespacedID, EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(currentTime)}})`)
**minimalCreateProfileInputTemplate for billing profile setup** — Tests that exercise billing call minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID()) (defined in framework_test.go) to get a baseline billing profile with AlignmentKindSubscription and PT0S collection interval. Override fields before calling billingService.CreateProfile. (`profInput := minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID()); _, err := tDeps.billingService.CreateProfile(ctx, profInput)`)
**Hardcoded test-namespace** — All tests use namespace := "test-namespace" — hardcoded in subscriptiontestutils.SetupDBDeps. Do not use any other namespace string. (`namespace := "test-namespace"`)
**Direct constructor wiring, no app/common imports** — framework_test.go builds billingservice, billingadapter, subscriptionsyncservice, appservice, etc. by calling their New(Config{...}) constructors directly. Importing app/common creates import cycles. (`billingService, err := billingservice.New(billingservice.Config{Adapter: billingAdapter, CustomerService: deps.CustomerService, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `framework_test.go` | Defines testDeps struct, setup() function, and minimalCreateProfileInputTemplate. All scenario files share this single test package (package subscription_test). Registers appsandbox.NewMockableFactory as side-effect before billing service use. | Adding a new service to setup() requires matching field in testDeps; forgetting cleanup(t) leaks the pgtestdb database; sandbox app must be registered via appsandbox.NewMockableFactory before billing wiring. |
| `scenario_editaligned_test.go` | Tests that editing a running subscription preserves the metered entitlement's current usage period boundaries after a PatchRemoveItem + PatchAddItem. | Assertions compare sView.Phases[0].ItemsByKey[key][0] (old/expired) vs sUpdated.Phases[0].ItemsByKey[key][1] (new replacement) — index 1 is the replacement entitlement. |
| `scenario_editcancel_test.go` | Tests editing then immediately canceling a subscription; also stress-tests with 10 parallel customers to surface concurrency issues. | Extra customers use unique subject keys (subject_2..subject_11); ensure all subject keys are unique per customer or customer service rejects duplicates. |
| `scenario_entinnextphase_test.go` | Regression test for subscription creation with a metered entitlement in the second phase (not the first). Does NOT create a billing profile — tests subscription correctness only. | 'THIS IS THE TEST, it used to fail' comment marks the key assertion. Do not add billing assertions here without first creating a profile. |
| `scenario_firstofmonth_test.go` | Tests billing anchor alignment: in-arrears usage, in-advance flat fee, and daily cadence ratecards align to first-of-month. Calls subscriptionSyncService.SynchronizeSubscription directly. | Line count depends on days between startOfSub and endOfMonth. Assertions group lines via lo.GroupBy and sort by period start — fragile to date changes in GetRFC3339Time values. |

## Anti-Patterns

- Importing app/common or any Wire provider set — creates import cycles with domain testutils packages.
- Using context.Background() where t.Context() is available — test harness lifecycle must own context cancellation.
- Using a namespace other than 'test-namespace' — hardcoded in SetupDBDeps infrastructure.
- Calling subscriptionSyncService or billingService without first creating a billing profile — profile is required for invoice generation.
- Mutating clock.Now() via clock.SetTime without restoring it after the test — leaked global clock state affects parallel tests.

## Decisions

- **Scenario files rather than table-driven tests.** — Each scenario exercises a different causal sequence of domain operations varying in setup, timing, and assertions; table-driven structure would obscure the temporal flow central to billing correctness.
- **All wiring done via direct constructors in framework_test.go instead of app/common Wire sets.** — Avoids import cycles between domain testutils and the application wiring layer; keeps test dependencies minimal and explicit.
- **Shared testDeps with embedded SubscriptionDependencies rather than per-test setup structs.** — Each test calls setup() independently (fresh pgtestdb DB), but the struct shape is shared so scenario files can access all services without re-declaring them.

## Example: Standard scenario: setup deps, control clock, create feature+plan+customer, subscribe, advance time, assert

```
func TestMyScenario(t *testing.T) {
	namespace := "test-namespace"
	currentTime := testutils.GetRFC3339Time(t, "2025-06-15T12:00:00Z")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	tDeps := setup(t, setupConfig{})
	defer tDeps.cleanup(t)
	clock.SetTime(currentTime)

	// Billing profile required for invoice generation
	_, err := tDeps.billingService.CreateProfile(ctx, minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID()))
	require.NoError(t, err)

	// 1st, create feature
// ...
```

<!-- archie:ai-end -->
