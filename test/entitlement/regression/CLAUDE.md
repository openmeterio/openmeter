# regression

<!-- archie:ai-start -->

> Integration regression test suite for the entitlement + credit engine, wiring real Ent/Postgres adapters against a MockStreamingConnector to reproduce and guard specific grant burn-down, expiry, voiding, and reset edge cases identified in production bugs.

## Patterns

**Self-contained bootstrap via setupDependencies** — Each test calls setupDependencies(t), which constructs every adapter and service from underlying package constructors (no app/common) and returns a Dependencies struct. Tests must defer deps.Close() to release DBClient, EntDriver, and PGDriver. (`deps := setupDependencies(t); defer deps.Close()`)
**Controlled clock with defer reset** — Every test calls defer clock.ResetTime() at the top, then drives time forward with clock.SetTime(testutils.GetRFC3339Time(t, "...")) before each operation. Skipping the defer leaks global clock state into parallel tests. (`defer clock.ResetTime(); clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:30:21Z"))`)
**MockStreamingConnector for usage events** — deps.Streaming (streamingtestutils.MockStreamingConnector) injects synthetic usage via AddSimpleEvent; never real ClickHouse. Future-dated events are deliberate to avoid 'event in future' connector errors. (`deps.Streaming.AddSimpleEvent("meter-1", 10, testutils.GetRFC3339Time(t, "2024-07-09T13:09:00Z"))`)
**No app/common imports in test wiring** — All adapters and services are built directly from their own constructors (customeradapter.New, customerservice.New, entitlementservice.NewEntitlementService, etc.) to keep helpers independent of the DI layer and avoid import cycles. (`customerAdapter, err := customeradapter.New(customeradapter.Config{Client: dbClient, Logger: log})`)
**createCustomerAndSubject helper** — Tests create a subject and matching customer together via createCustomerAndSubject(...) before creating entitlements; the helper creates the subject first, then a customer with matching SubjectKeys. (`cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")`)
**Hooks registered after construction** — entitlementsubscriptionhook and credithook are registered on connectors via RegisterHooks after construction, mirroring the production wiring order in app/common. (`entitlementConnector.RegisterHooks(entitlementsubscriptionhook.NewEntitlementSubscriptionHook(...), credithook.NewEntitlementHook(grantRepo))`)
**Scenario tests named after the bug** — Test function names describe the exact production edge case (e.g. TestGrantExpiringAtReset, TestBalanceCalculationsAfterVoiding) and inline comments document the originally faulty behaviour and expected output. (`assert.Equal(0.0, currentBalance.Balance) // previously faulty: grant2 has expired by this point`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `framework_test.go` | Defines the Dependencies struct, setupDependencies(t) (the single wiring function using real Postgres via testutils.InitPostgresDB + mock streaming), and createCustomerAndSubject. deps.Close() releases DBClient, EntDriver, PGDriver. | dbClient.Schema.Create uses context.Background() (pre-existing — not a model to copy). Adding a dependency requires extending both the struct fields and the return literal. |
| `scenario_test.go` | All scenario-named regression tests. Each reproduces a production bug timeline with exact RFC3339 timestamps, sequential clock.SetTime calls, and deterministic streaming events. | Do not normalise or truncate the exact timestamps; future-timestamped streaming events are deliberate connector-error workarounds, not mistakes. |

## Anti-Patterns

- Importing app/common or any Wire provider set — test wiring must stay independent to avoid import cycles
- Using context.Background() in new test code where t.Context() is available
- Reusing the same subject/feature key across tests without isolated setupDependencies calls
- Adding business logic or helpers beyond test scaffolding — this is purely a regression harness
- Skipping defer clock.ResetTime() — leaves global clock state dirty for parallel tests

## Decisions

- **Wire from package constructors, not app/common** — Keeps test helpers isolated from application DI so unrelated wiring changes cannot introduce import cycles or break this package.
- **MockStreamingConnector instead of real ClickHouse** — Regression tests need deterministic usage values at exact timestamps without ClickHouse infrastructure and without timing flakiness.
- **Scenario tests named after the bug they reproduce** — Makes the guarded production edge case obvious; the inline comment documents the original faulty behaviour so the regression intent survives refactors.

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
