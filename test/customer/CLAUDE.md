# customer

<!-- archie:ai-start -->

> Integration tests for customer, subject, subscription, entitlement, plan, and billing domains wired together — validates cross-domain lifecycle correctness (usage attributions, subject hooks, subscription constraints, invoice generation) against real PostgreSQL.

## Patterns

**TestEnv interface with all cross-domain services** — TestEnv exposes Customer(), Subscription(), SubscriptionWorkflow(), Entitlement(), Feature(), Subject(), Plan(), Billing(), App(), and Meter() methods so test methods can exercise multi-domain flows without constructing services inline. (`type TestEnv interface { Customer() customer.Service; Subject() subject.Service; SubscriptionWorkflow() subscriptionworkflow.Service; Billing() billing.Service; ... }`)
**Hooks registered after service construction in NewTestEnv** — After building customerService and subjectService from raw constructors, cross-domain hooks are wired manually: customerService.RegisterRequestValidator(entValidator), subjectService.RegisterHooks(subjectCustomerHook), customerService.RegisterHooks(customerSubjectHook), customerService.RegisterRequestValidator(subsCustValidator). Order matters. (`customerService.RegisterRequestValidator(entcustomervalidator.NewValidator(entitlementRegistry.EntitlementRepo))
subjectService.RegisterHooks(subjectCustomerHook)
customerService.RegisterHooks(customerSubjectHook)`)
**subscriptiontestutils.SetupDBDeps for DB bootstrapping** — NewTestEnv calls subscriptiontestutils.SetupDBDeps(t) instead of calling testutils.InitPostgresDB directly. This handles schema migration and returns a DBClient ready for all adapters in one call. (`dbDeps := subscriptiontestutils.SetupDBDeps(t)`)
**Unique ULID namespace per test method via setupNamespace** — Every test method in CustomerHandlerTestSuite begins with s.setupNamespace(t), which assigns a new ulid.Make().String() to s.namespace, providing full test isolation on the shared database. (`func (s *CustomerHandlerTestSuite) setupNamespace(t *testing.T) {
	s.namespace = ulid.Make().String()
}`)
**clock.SetTime + t.Cleanup(clock.ResetTime) for time-sensitive tests** — Tests that need to advance time call clock.SetTime(future) and always register t.Cleanup(clock.ResetTime) to prevent global clock state from leaking between tests. (`clock.SetTime(future)
t.Cleanup(clock.ResetTime)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Constructs the full multi-domain service graph from raw package constructors. Wires meter, customer, entitlement, subject, plan, app, subscription, workflow, and billing. Registers all hooks post-construction. Exposes TestEnv interface. | Hook registration order matters: entValidator before subjectHook before subscriptionValidator. Never import app/common. noopCustomerOverrideService is an inline noop for billing.CustomerOverrideService — do not replace with a real implementation. |
| `customer.go` | CustomerHandlerTestSuite with CRUD tests for customer lifecycle, key conflict detection, and subscription-present update constraints. | UsageAttribution is nil (not empty slice) when no subjects are attached — assertions must check Nil not Empty. TestKey is reused across test methods only because setupNamespace provides namespace isolation. |
| `subject.go` | Tests subject deletion edge cases (dangling subjects, subjects still attached to customers, subjects with active entitlements) and TestMultiSubjectIntegrationFlow — a full plan+subscription+invoice end-to-end. | TestMultiSubjectIntegrationFlow installs a sandbox app and creates a billing profile — both must succeed or the test panics. clock.ResetTime must be deferred immediately after clock.SetTime. |
| `customer_test.go` | Entry point: creates TestEnv, defers Close(), runs Customer and Subject sub-test groups via CustomerHandlerTestSuite and SubjectHandlerTestSuite. | t.Errorf used for Close errors (not t.Fatal) to ensure deferred Close always runs even if sub-tests fail. |

## Anti-Patterns

- Importing app/common or Wire provider sets — all wiring must use raw package constructors to avoid import cycles.
- Using context.Background() in new test code where t.Context() is available.
- Reusing the same subject key or customer key across test functions without setupNamespace — causes unique-constraint violations.
- Forgetting to defer clock.ResetTime() after clock.SetTime() — leaves global clock dirty for subsequent tests.
- Asserting UsageAttribution as empty slice instead of nil — service returns nil when there are no subject keys.

## Decisions

- **TestEnv exposes all cross-domain services (10 methods) rather than just customer.** — Subject deletion, subscription constraints, and multi-subject billing flows require real instances of subscription, entitlement, billing, and plan — a minimal customer-only env cannot exercise these paths.
- **Hooks wired manually in NewTestEnv rather than discovered via DI.** — Tests must stay independent from app/common. Manual hook registration ensures the exact hook set matches production wiring without importing Wire providers or creating import cycles.
- **subscriptiontestutils.SetupDBDeps used for DB bootstrap.** — Centralises pgtestdb provisioning and schema migration so multiple test packages share the same proven setup path, reducing per-package boilerplate.

## Example: Multi-domain integration test: generate unique namespace, create customer with subject, subscribe to a plan, assert subscription is active.

```
s.setupNamespace(t)
ctx := context.Background()
app := s.installSandboxApp(t, s.namespace)
_ = s.createDefaultProfile(t, app, s.namespace)
cust, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
	Namespace: s.namespace,
	CustomerMutate: customer.CustomerMutate{
		Name: "Test",
		UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"subj-1"}},
	},
})
require.NoError(t, err)
sub, err := s.Env.SubscriptionWorkflow().CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
	Namespace: s.namespace, CustomerID: cust.ID,
})
// ...
```

<!-- archie:ai-end -->
