# regression

<!-- archie:ai-start -->

> Integration regression test suite for the entitlement + credit engine, wiring real Ent/Postgres adapters against a MockStreamingConnector to reproduce and guard specific grant burn-down, expiry, voiding, and reset edge cases identified in production bugs.

## Patterns

**Self-contained dependency bootstrap via setupDependencies** — Each test calls setupDependencies(t) which constructs every adapter and service from underlying package constructors (no app/common), returning a Dependencies struct. Tests must call deps.Close() via defer to release DB and driver connections. (`deps := setupDependencies(t); defer deps.Close()`)
**Controlled clock advancement with defer reset** — Every test must call defer clock.ResetTime() at the top, then drives time forward with clock.SetTime(testutils.GetRFC3339Time(t, "...")) before each operation to reproduce a precise temporal scenario. Without defer clock.ResetTime(), global clock state leaks into other tests. (`defer clock.ResetTime(); clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:30:21Z"))`)
**MockStreamingConnector for usage events** — deps.Streaming (streamingtestutils.MockStreamingConnector) is used to inject synthetic usage events via AddSimpleEvent; never real ClickHouse. Future-dated events are a deliberate hack to avoid 'event in future' errors from the connector. (`deps.Streaming.AddSimpleEvent("meter-1", 10, testutils.GetRFC3339Time(t, "2024-07-09T13:09:00Z"))`)
**No app/common imports in test wiring** — All adapters and services are instantiated directly from their own constructors (e.g. customeradapter.New, customerservice.New, entitlementservice.NewEntitlementService) to keep test helpers independent from the DI layer and avoid import cycles. (`customerAdapter, err := customeradapter.New(customeradapter.Config{Client: dbClient, Logger: log})`)
**createCustomerAndSubject helper for test subjects** — Tests create both a subject and customer together via createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, ns, key, name) before creating entitlements. The helper creates the subject first, then a customer with matching SubjectKeys. (`cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")`)
**Hooks registered on connectors after construction** — entitlementsubscriptionhook and credithook are registered on connectors after construction via RegisterHooks, mirroring the production wiring order in app/common. Both meteredEntitlementConnector and entitlementConnector receive their hooks after being fully constructed. (`entitlementConnector.RegisterHooks(entitlementsubscriptionhook.NewEntitlementSubscriptionHook(...), credithook.NewEntitlementHook(grantRepo))`)
**Scenario tests named after the bug they reproduce** — Test function names describe the exact production edge case (e.g. TestGrantExpiringAtReset, TestBalanceCalculationsAfterVoiding). Comments inside each test document the originally faulty behaviour and the expected correct output. (`assert.Equal(0.0, currentBalance.Balance) // This test was previously faulty, grant2 has expired by this point`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `framework_test.go` | Defines Dependencies struct and setupDependencies(t) — the single wiring function that builds all adapters and services using real Postgres (via testutils.InitPostgresDB) and mock streaming. Also defines createCustomerAndSubject helper. | dbClient.Schema.Create uses context.Background() (pre-existing, not a model to copy in new code). Adding a new dependency requires extending both the Dependencies struct fields and the return literal. deps.Close() releases DBClient, EntDriver, and PGDriver. |
| `scenario_test.go` | Contains all scenario-named regression tests. Each test reproduces a specific production bug timeline using exact RFC3339 timestamps, sequential clock.SetTime calls, and deterministic streaming events. | Tests rely on exact RFC3339 timestamps to reproduce expiry/reset edge cases — do not normalise or truncate times. Future-timestamped streaming events (e.g. 2025-...) are deliberate hacks to avoid connector errors, not mistakes. Assert comments document previously broken behaviour. |

## Anti-Patterns

- Importing app/common or any Wire provider set — test wiring must stay independent to avoid import cycles
- Using context.Background() in new test code where t.Context() is available (framework_test.go has pre-existing uses that are not models to follow)
- Reusing the same subject key or feature key across test functions without isolated setupDependencies calls — each test gets its own DB via pgtestdb isolation
- Adding business logic or helper functions beyond test scaffolding — this package is purely a regression harness for specific bugs
- Skipping defer clock.ResetTime() — leaves global clock state dirty for other tests running in parallel

## Decisions

- **Wire from package constructors, not app/common** — Keeps test helpers isolated from application DI so unrelated wiring changes cannot introduce import cycles or break this test package.
- **MockStreamingConnector instead of real ClickHouse** — Regression tests need deterministic usage values at exact timestamps; MockStreamingConnector provides this without ClickHouse infrastructure and eliminates timing flakiness.
- **Scenario tests named after the bug they reproduce** — Makes it obvious which production edge case each test guards; the comment inside each test documents the original faulty behaviour so the regression intent survives code changes.

## Example: Minimal new regression test structure

```
func TestMyRegressionCase(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := t.Context()
	assert := assert.New(t)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-01T00:00:00Z"))
	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")
	// create feature, entitlement, grants, add streaming events, assert balance
	_ = cust
	_ = assert
}
```

<!-- archie:ai-end -->
