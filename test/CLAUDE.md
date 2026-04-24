# test

<!-- archie:ai-start -->

> Integration and cross-domain test suites that require a real PostgreSQL database. Each child owns one domain or cross-domain scenario; all wire services from raw package constructors (never app/common) to avoid import cycles. This folder is the canonical home for tests that span multiple openmeter/* packages.

## Patterns

**BaseSuite embedding for shared billing infrastructure** — Tests that need the full billing stack embed test/billing.BaseSuite (or extend it with SubscriptionMixin). BaseSuite provisions Atlas migrations, a unique namespace, MockStreamingConnector, and pkg/clock control. (`type MyTestSuite struct { billingtest.BaseSuite } — then access s.BillingService, s.MockStreamingConnector.`)
**TestEnv interface for service access** — test/notification, test/app, and test/customer expose a TestEnv interface rather than service structs directly, keeping the wiring details hidden from test methods. (`env := notification.NewTestEnv(t); defer env.Close(); env.Notification().CreateChannel(...)`)
**Unique namespace per test method** — Each test method generates a unique namespace (ULID or function-name-based) to prevent FK constraint violations when tests share a database. (`ns := ulid.Make().String(); // or setupNamespace(t) returning a unique string`)
**Raw package constructors only — no app/common imports** — All test wiring constructs services from their direct constructors (adapter.New, service.New) not from Wire provider sets in app/common to avoid import cycles. (`adapter := billingadapter.New(db); svc := billingservice.New(adapter, ...)`)
**clock.SetTime + defer clock.ResetTime for deterministic time** — Tests that depend on time call pkg/clock.SetTime at the start and defer clock.ResetTime() to restore global state for parallel tests. (`clock.SetTime(t0); defer clock.ResetTime(); // then call service methods`)
**t.Context() over context.Background()** — All new test code uses t.Context() (or s.T().Context()) so context cancellation is tied to the test harness lifecycle. (`ctx := t.Context() // not context.Background()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `test/billing/suite.go` | BaseSuite: provisions Atlas migrations, unique namespace, MockStreamingConnector, billing service stack. Extended by most billing tests. | Always use BaseSuite rather than manually provisioning these — duplicating setup leads to subtle test isolation bugs. |
| `test/billing/subscription_suite.go` | SubscriptionMixin: adds subscription+plan wiring on top of BaseSuite for tests that exercise billing+subscription flows. | Use SubscriptionMixin when a test needs both billing and subscription services; do not wire subscription separately. |
| `test/notification/testenv.go` | Wires the full notification stack including a real Svix client (requires SVIX_HOST env var). | Do not create a second TestEnv inside a test method — each starts goroutines and DB connections. Always defer env.Close(). |
| `test/credits/sanity_test.go` | Entry point for credits/charges integration tests; uses chargestestutils.NewServices as the canonical charges stack builder. | Use createLedgerBackedCustomer for tests requiring ledger accounts, not BaseSuite.CreateTestCustomer. |
| `test/subscription/framework_test.go` | Wires all subscription+billing deps via direct constructors and subscriptiontestutils.SetupDBDeps. Shared testDeps struct used by all scenario files. | Hardcoded namespace 'test-namespace'. Do not use a different namespace in this package. |
| `test/app/testenv.go` | TestEnv for app integration tests; wires billing service via InitBillingService helper without app/common. | Always call setupNamespace(t) per test method in AppHandlerTestSuite — never share a namespace across methods. |

## Anti-Patterns

- Importing app/common or any Wire provider set in test files — always wire from raw package constructors to avoid import cycles
- Using context.Background() where t.Context() or s.T().Context() is available
- Reusing the same namespace, subject key, or customer key across test functions in the same package
- Omitting defer clock.ResetTime() after clock.SetTime() — leaves global clock dirty for subsequent parallel tests
- Accessing adapter methods directly when the service covers the same operation — bypasses state machine and validation

## Decisions

- **Integration tests live in test/* rather than co-located with production code.** — Cross-domain tests import multiple openmeter/* packages; placing them in a neutral test/* tree prevents import cycles and makes the cross-domain dependency explicit.
- **Atlas migrations run by default in BaseSuite (not ent.Schema.Create).** — Running Atlas migrations in tests validates migration files against the live schema, catching drift before CI. TEST_DISABLE_ATLAS exists as an escape hatch only.
- **Scenario-based test files in test/subscription rather than table-driven tests.** — Subscription lifecycle scenarios are complex multi-step flows where numbered step comments provide clearer documentation than a flat table of inputs/outputs.

## Example: Extending BaseSuite for a billing+subscription integration test

```
package mytest

import (
	"testing"

	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type Suite struct {
	billingtest.BaseSuite
	billingtest.SubscriptionMixin
}

func TestMySuite(t *testing.T) {
	ts := &Suite{}
// ...
```

<!-- archie:ai-end -->
