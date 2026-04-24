# subscription

<!-- archie:ai-start -->

> Integration test suite for subscription lifecycle scenarios — verifying the full billing+sync stack (plan creation, subscription create/edit/cancel, billing sync, invoice line generation) against a real PostgreSQL database via pgtestdb. Tests are scenario-based and exercise cross-domain flows that unit tests cannot cover.

## Patterns

**Single shared setup() function via testDeps** — All tests call setup(t, setupConfig{}) which wires every dependency from scratch using direct constructors (no app/common imports). The returned testDeps struct embeds subscriptiontestutils.SubscriptionDependencies and adds billing, sync, and app services. (`tDeps := setup(t, setupConfig{}); defer tDeps.cleanup(t)`)
**clock.SetTime for deterministic time control** — Tests set a fixed RFC3339 start time via testutils.GetRFC3339Time then advance it with clock.SetTime(currentTime.Add(...)). All services use pkg/clock so this controls all time-dependent billing logic. (`currentTime := testutils.GetRFC3339Time(t, "2025-01-20T13:11:07Z"); clock.SetTime(currentTime)`)
**Numbered step comments for scenario readability** — Test bodies use numbered comments (// 1st, // 2nd, etc.) to document the causal sequence: feature creation → plan creation → publish → customer creation → subscription creation → mutations → assertions. (`// 1st, let's create the features ... // 2nd, let's create the plan ... // 4th, let's create the subscription`)
**Plan published before subscription creation** — Every scenario calls PlanService.PublishPlan with EffectiveFrom before creating a subscription via pcSubscriptionService.Create. An unpublished plan will cause the subscription creation to fail. (`p, err = tDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{NamespacedID: p.NamespacedID, EffectivePeriod: ...})`)
**minimalCreateProfileInputTemplate for billing profile setup** — Tests that exercise billing use the shared minimalCreateProfileInputTemplate(appID) helper defined in framework_test.go with AlignmentKindSubscription and PT0S collection interval. Override fields as needed before calling billingService.CreateProfile. (`profInput := minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID()); profInput.WorkflowConfig.Collection.Alignment = billing.AlignmentKindAnchored`)
**Hardcoded test-namespace** — All tests use namespace := "test-namespace" — this is hardcoded in subscriptiontestutils.SetupDBDeps. Do not use a different namespace string or test infrastructure will break. (`namespace := "test-namespace"`)
**Direct constructor wiring, no app/common imports** — framework_test.go builds billingservice, billingadapter, subscriptionsyncservice, appservice, etc. by calling their New(Config{...}) constructors directly. Importing app/common would create import cycles with testutils. (`billingService, err := billingservice.New(billingservice.Config{Adapter: billingAdapter, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `framework_test.go` | Defines testDeps struct, setup() function, and minimalCreateProfileInputTemplate helper. All scenario files share this single test package (package subscription_test) and depend on this file. | Adding a new service to setup() requires a matching field in testDeps; forgetting cleanup(t) leaks the pgtestdb database; the sandbox app must be registered via appsandbox.NewMockableFactory before being used in billingService wiring. |
| `scenario_editaligned_test.go` | Tests that editing a running subscription preserves the metered entitlement's current usage period boundaries (cadence alignment is maintained after remove+add item patch). | Assertions compare sView.Phases[0].ItemsByKey[key][0] (old) vs sUpdated.Phases[0].ItemsByKey[key][1] (new) — index 1 is the replacement entitlement after the edit; index 0 is the expired one. |
| `scenario_editcancel_test.go` | Tests editing a subscription then immediately canceling it; also stress-tests with 10 parallel customers/subscriptions to surface concurrency issues. | Extra customers are created in a loop with unique subject keys (subject_2..subject_11); ensure subject keys are unique per customer or the customer service will reject duplicates. |
| `scenario_entinnextphase_test.go` | Regression test for a bug where creating a subscription with a metered entitlement in the second phase (not the first) would fail. The comment 'THIS IS THE TEST, it used to fail' marks the key assertion. | Does NOT create a billing profile — tests only subscription creation correctness, not invoice generation. |
| `scenario_firstofmonth_test.go` | Tests billing anchor alignment: billing lines for in-arrears usage, in-advance flat fee, and daily cadence ratecards are verified to align to the first-of-month anchor. Also tests early-cancel + sync-past-anchor produces correct line. | Uses subscriptionSyncService.SynchronizeSubscription directly to trigger invoice line generation. Assertions group lines by feature key via lo.GroupBy and sort by period start — line count depends on days between startOfSub and endOfMonth. |

## Anti-Patterns

- Importing app/common or any Wire provider set — creates import cycles with domain testutils packages
- Using context.Background() where t.Context() is available (test harness lifecycle must own context cancellation)
- Using a namespace other than 'test-namespace' — hardcoded in SetupDBDeps infrastructure
- Calling subscriptionSyncService or billingService without first creating a billing profile — profile is required for invoice generation
- Mutating clock.Now() without restoring it after the test — leaked global clock state affects parallel tests

## Decisions

- **Scenario files rather than table-driven tests** — Each scenario exercises a different causal sequence of domain operations that varies in setup, timing, and assertions; a table-driven structure would obscure the temporal flow that is central to billing correctness.
- **All wiring done via direct constructors in framework_test.go instead of app/common Wire sets** — Avoids import cycles between domain testutils and the application wiring layer; keeps test dependencies minimal and explicit so unrelated wiring changes cannot silently affect these tests.
- **Shared testDeps with embedded SubscriptionDependencies rather than per-test setup** — Each test calls setup() independently (getting a fresh pgtestdb database), but the struct shape is shared so scenario files can access all services without re-declaring them.

## Example: Standard scenario structure: setup deps, control clock, create feature+plan+customer, create subscription, advance time, assert

```
func TestMyScenario(t *testing.T) {
	namespace := "test-namespace"
	currentTime := testutils.GetRFC3339Time(t, "2025-06-15T12:00:00Z")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tDeps := setup(t, setupConfig{})
	defer tDeps.cleanup(t)
	clock.SetTime(currentTime)

	// Create billing profile (required for invoice generation)
	_, err := tDeps.billingService.CreateProfile(ctx, minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID()))
	require.NoError(t, err)

	// Create feature, plan, publish plan, create customer, then subscription
// ...
```

<!-- archie:ai-end -->
