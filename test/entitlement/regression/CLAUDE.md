# regression

<!-- archie:ai-start -->

> Regression test suite that exercises the full metered-entitlement + credit-grant stack (feature, entitlement, grant, balance snapshot, reset, void) end-to-end against a real Postgres DB, reproducing historically faulty balance-calculation scenarios. Its primary constraint: tests wire the production connectors directly and drive time via clock so balance/reset/expiry interactions are deterministic.

## Patterns

**Real-stack assembly via setupDependencies** — Each test calls setupDependencies(t), which builds the entire entitlement/credit graph from concrete constructors (feature, grant, balance snapshot, metered/static/boolean connectors, customer, subject) against a fresh Postgres DB from testutils.InitPostgresDB(t). Build from package constructors, never from app/common wiring. (`deps := setupDependencies(t); defer deps.Close()`)
**Deterministic time via clock + frozen RFC3339** — Every temporal step uses clock.SetTime(testutils.GetRFC3339Time(t, "...")) to advance the simulated clock, with defer clock.ResetTime() at the top of each test. Balances are queried at explicit timestamps; reset/expiry math depends on these exact times. (`defer clock.ResetTime(); clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:30:21Z"))`)
**Usage injected through MockStreamingConnector** — Usage events are added via deps.Streaming.AddSimpleEvent("meter-1", value, time). The meter is named meter-1 with MeterAggregationCount; events placed far in the future are a deliberate hack to avoid streaming errors while keeping them out of the queried window. (`deps.Streaming.AddSimpleEvent("meter-1", 10, testutils.GetRFC3339Time(t, "2024-07-09T13:09:00Z"))`)
**Customer+subject created together before entitlement** — createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, ns, key, name) creates a subject then a customer with UsageAttribution SubjectKeys=[key]; the returned customer's GetUsageAttribution() feeds CreateEntitlementInputs.UsageAttribution. (`cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")`)
**Hooks registered on connectors after construction** — meteredEntitlementConnector.RegisterHooks(...) and entitlementConnector.RegisterHooks(...) wire the subscription hook and credit EntitlementHook, mirroring production wiring so reset/void side effects fire. (`entitlementConnector.RegisterHooks(entitlementsubscriptionhook.NewEntitlementSubscriptionHook(...), credithook.NewEntitlementHook(grantRepo))`)
**Balance asserted by exact float at a timestamp** — Tests call deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace, ID}, at) and assert.Equal an exact float (e.g. 30.0, 0.0, 488.0). Comments document why the expected value holds (grant priority, expiry, reset rollover). (`assert.Equal(488.0, currentBalance.Balance)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `framework_test.go` | Test harness: Dependencies struct, Close(), setupDependencies(t) wiring the full credit+entitlement stack, and createCustomerAndSubject helper. Package framework_test. | Uses slog.Default() and context.Background() and eventbus.NewMock(t) — acceptable in this test-only harness but do not copy into production constructors. meter-1 / namespace-1 / Meter1ID are hardcoded; all entitlements must target namespace-1. |
| `scenario_test.go` | The actual regression cases: TestGrantExpiringAtReset, TestGrantExpiringAndRecurringAtReset, TestBalanceCalculationsAfterVoiding, TestCreatingEntitlementsForKeyOfArchivedFeatures. Each is a timed sequence of feature/entitlement/grant/reset/void calls with balance assertions. | Expected balances encode subtle grant-priority/expiry/reset-rollover semantics; comments (e.g. 'This test was previously faulty') flag historical bugs — do not 'fix' an assertion to make it pass without understanding the documented reason. |

## Anti-Patterns

- Importing app/common or production DI wiring instead of building deps from package constructors — creates test-only import cycles and defeats the regression intent.
- Using time.Now() or real sleeps instead of clock.SetTime + frozen RFC3339 timestamps — makes reset/expiry math nondeterministic.
- Omitting defer clock.ResetTime() / defer deps.Close() — leaks frozen time and DB resources into other tests.
- Changing an asserted balance float to silence a failure without reconciling it against the grant priority / expiry / rollover comments.
- Adding usage with AddSimpleEvent inside the queried window when the scenario intends it to be ignored (future-dated hack).

## Decisions

- **Assemble production connectors directly against a real Postgres DB rather than mocking the credit/entitlement engine.** — These are regression tests for balance-calculation bugs that only surface through the real grant/snapshot/reset interaction; mocks would hide them.
- **Drive all time through pkg/clock with hardcoded RFC3339 instants.** — Entitlement usage periods, grant expiration, and reset anchors are time-sensitive; deterministic clock control makes the exact expected balances reproducible.

## Example: Standard regression test skeleton: timed feature/entitlement/grant setup then exact-balance assertion.

```
func TestX(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:30:21Z"))
	feature, _ := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{Name: "feature-1", Key: "feature-1", Namespace: "namespace-1", MeterID: convert.ToPointer(deps.Meter1ID)})
	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")
	entitlement, _ := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{Namespace: "namespace-1", FeatureID: &feature.ID, FeatureKey: &feature.Key, UsageAttribution: cust.GetUsageAttribution(), EntitlementType: entitlement.EntitlementTypeMetered, UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{Interval: timeutil.RecurrencePeriodDaily, Anchor: testutils.GetRFC3339Time(t, "2024-06-28T14:48:00Z")}))}, nil)
	bal, _ := deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx, models.NamespacedID{Namespace: "namespace-1", ID: entitlement.ID}, testutils.GetRFC3339Time(t, "2024-06-28T14:36:45Z"))
	assert.Equal(30.0, bal.Balance)
}
```

<!-- archie:ai-end -->
