# testutils

<!-- archie:ai-start -->

> Shared test fixtures and dependency builders for the subscription domain. Provides SubscriptionDependencies (a fully wired set of real services from Ent adapters), example entities, comparison helpers, and mocks. Must NOT import app/common to avoid import cycles.

## Patterns

**NewService builds full dependency graph from package constructors** — NewService(t, dbDeps) wires all real adapters and services (customer, subject, entitlement, plan, addon, subscription, workflow) from their underlying constructors, not from app/common Wire providers, to avoid import cycles. (`deps := subscriptiontestutils.NewService(t, dbDeps)`)
**SetupDBDeps provisions a real Postgres DB with full migration** — SetupDBDeps(t) calls testutils.InitPostgresDB, runs migrate.Up(), and returns DBDeps{DBClient, EntDriver, PGDriver}. Tests must call deps.Cleanup(t) via t.Cleanup. (`dbDeps := subscriptiontestutils.SetupDBDeps(t)
t.Cleanup(func() { dbDeps.Cleanup(t) })`)
**Example* vars as canonical test data** — ExampleRateCard1–5, ExampleAddonRateCard1–6, ExampleFeatureKey/2/3, ExampleNamespace, ISOMonth are package-level vars. Always clone rate cards before using as plan/subscription input to avoid test-to-test mutation. (`b.AddPhase(lo.ToPtr(ISOMonth), ExampleRateCard1.Clone())`)
**testAddonService wraps addon.Service with CreateTestAddon helper** — CreateTestAddon creates and immediately publishes the addon, which is the required two-step flow. Use deps.AddonService.CreateTestAddon(t, inp) rather than calling service.CreateAddon then PublishAddon separately. (`add := deps.AddonService.CreateTestAddon(t, addonInp)`)
**MockService and MockWorkflowService for unit tests** — MockService satisfies subscription.Service; MockWorkflowService satisfies subscriptionworkflow.Service. Wire Fn fields for the methods under test; leave others nil to get panics on unexpected calls. (`svc := &subscriptiontestutils.MockService{GetViewFn: func(...) {...}}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | NewService — the canonical wiring entry point for subscription integration tests. SubscriptionDependencies struct holds all assembled services. | After SetDBClient, read back meterID via GetMeterByIDOrSlug — pgtestdb template reuse can resolve to a different ID. |
| `db.go` | SetupDBDeps with mutex-guarded Postgres provisioning and full migration. | sync.Mutex guards serial DB setup; parallel subtests must each call SetupDBDeps independently. |
| `compare.go` | ValidateSpecAndView deep-compares a SubscriptionSpec against a SubscriptionView including entitlement alignment, feature linking, and phase cadences. | UsagePeriod alignment checks use Truncate(time.Minute) — avoid sub-minute time precision in test fixtures. |
| `helpers.go` | CreatePlanWithAddon, CreateSubscriptionFromPlan, CreateAddonForSubscription — high-level test scenario builders. | CreateAddonForSubscription encodes a Quantity=0 terminal entry to represent a closed cadence; use CreateMultiInstanceAddonForSubscription for multi-instance addons. |
| `builder.go` | testPlanbuilder and testSubscriptionSpecBuilder fluent DSLs for plan and spec construction in tests. | testSubscriptionSpecBuilder.Build() calls SyncAnnotations and Validate — always use Build() rather than using the spec directly. |
| `mock.go` | MockService and MockWorkflowService with Fn fields. | MockWorkflowService does not implement Restore, AddAddon, or ChangeAddonQuantity — calling these will panic. |

## Anti-Patterns

- Importing app/common from testutils — causes import cycles and violates the boundary that test deps must be built from package constructors.
- Using ExampleRateCard* vars directly as plan input without Clone() — shared package-level vars will be mutated across tests.
- Calling service.CreateAddon without immediately calling PublishAddon — use CreateTestAddon which does both.
- Constructing SubscriptionDependencies manually instead of via NewService — misses hook registration (annotationCleanupHook, entitlementValidatorHook) which are required for correct test semantics.
- Using context.Background() instead of t.Context() in test helpers.

## Decisions

- **All services built from package constructors, not app/common Wire providers.** — Prevents import cycles; keeps test wiring independent from production DI changes.
- **ValidateSpecAndView validates entitlement alignment against billingAnchor.** — Entitlement UsagePeriod must be anchored to subscription BillingAnchor — this is a billing correctness invariant that tests must enforce.

## Example: Set up a full subscription integration test with plan and addon

```
import subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"

func TestMyFeature(t *testing.T) {
	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	t.Cleanup(func() { dbDeps.Cleanup(t) })
	deps := subscriptiontestutils.NewService(t, dbDeps)

	plan, addon := subscriptiontestutils.CreatePlanWithAddon(t, deps,
		subscriptiontestutils.GetExamplePlanInput(t),
		subscriptiontestutils.GetExampleAddonInput(t, period),
	)
	subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, plan, clock.Now())
	subscriptiontestutils.ValidateSpecAndView(t, expectedSpec, subView)
}
```

<!-- archie:ai-end -->
