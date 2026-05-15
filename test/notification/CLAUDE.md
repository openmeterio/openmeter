# notification

<!-- archie:ai-start -->

> Integration test suite for the notification domain: channels, rules, events, delivery status, webhook delivery via Svix, and balance-threshold consumer handler. Wires the full notification stack including a real Svix client from environment variables via TestEnv.

## Patterns

**TestEnv interface for service access** — All test suites receive a TestEnv value and call env.Notification(), env.NotificationRepo(), env.Feature(), env.Meter(), env.NotificationWebhook() instead of accessing service structs directly. TestEnv is created via NewTestEnv(t, ctx, namespace) and must be closed via env.Close(). (`env, err := NewTestEnv(t, ctx, namespace); service := env.Notification()`)
**Sub-suite structs with Setup + named test methods** — Test behaviour is split into XxxTestSuite structs (ChannelTestSuite, RuleTestSuite, EventTestSuite, RepositoryTestSuite, WebhookTestSuite, BalanceNotificaiontHandlerTestSuite) each with an optional Setup method. The top-level TestNotification function constructs, sets up, and runs subtests. (`testSuite := ChannelTestSuite{Env: env}; t.Run("Create", func(t *testing.T) { testSuite.TestCreate(ctx, t) })`)
**Svix from SVIX_HOST env var** — testenv.go reads SVIX_HOST (defaults to 127.0.0.1) and SVIX_JWT_SECRET to connect to a real Svix instance. Webhook delivery assertions require docker compose up -d svix or SVIX_HOST set. (`svixHost := defaultx.IfZero(os.Getenv("SVIX_HOST"), DefaultSvixHost)`)
**NewBalanceSnapshotEvent for balance consumer tests** — BalanceNotificaiontHandlerTestSuite uses NewBalanceSnapshotEvent(BalanceSnapshotEventInput{Feature, Value, Namespace}) to construct synthetic snapshot.SnapshotEvent values passed to handler.Handle(ctx, event). (`snapshotEvent := NewBalanceSnapshotEvent(BalanceSnapshotEventInput{Feature: s.feature, Value: snapshot.EntitlementValue{Balance: convert.ToPointer(50.0)}, Namespace: s.namespace})`)
**setupNamespace for isolated balance handler tests** — BalanceNotificaiontHandlerTestSuite.setupNamespace creates a fresh ULID namespace, replaces meters, creates feature/channel/rule, and constructs consumer.EntitlementSnapshotHandler. Called inside each TestXxx method, not a shared Setup. (`s.setupNamespace(ctx, t); err := s.handler.Handle(ctx, snapshotEvent)`)
**eventHandler started as goroutine, closed via env.Close()** — testenv.go starts eventHandler.Start() in a goroutine inside NewTestEnv. The closerFunc calls eventHandler.Close() then closes Ent and PG drivers. Always defer env.Close() or use t.Cleanup. (`go func() { _ = eventHandler.Start() }(); t.Cleanup(func() { env.Close() })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Boots the full notification stack: Ent adapter, Svix webhook handler (real Svix client), pglockx lock client, notification event handler goroutine, notification service, feature connector, and meter mock adapter. Returns closerFunc for cleanup. | eventHandler.Start() runs in a goroutine — must call env.Close(). Svix connectivity is required for webhook delivery tests. TestEnv fields are set only after NewTestEnv returns. |
| `consumer_balance.go` | BalanceNotificaiontHandlerTestSuite and helpers. Tests multi-step threshold crossing and reset deduplication logic of the balance consumer handler via direct handler.Handle calls (not Kafka). | TestEntitlementCurrentUsagePeriod is a package-level var computed at init (time.Now()); tests depending on exact period boundaries may be flaky near boundaries. |
| `notification_test.go` | Top-level TestNotification entry point: creates one TestEnv, constructs all sub-suites, calls Setup on those that need it, runs subtests. | All sub-suites share the same TestEnv and namespace — test data created in one suite is visible to others. Use distinct IDs/names. |
| `repository.go` | RepositoryTestSuite tests repo-layer filter methods (ListEvents with Features/Subjects filters). Calls repo directly via env.NotificationRepo() to bypass service layer for precise filter testing. | Uses context.Background() in TestFilterEventByFeature/TestFilterEventBySubject — acceptable here since these are pure repo-filter tests without Ent tx propagation needs. |
| `helpers.go` | Stateless helpers: NewSvixAuthToken (generates HS256 JWT for Svix API auth) and NewClickhouseClient (present but currently unused by notification tests). | NewClickhouseClient exists for potential future use; notification tests do not currently use ClickHouse directly. |

## Anti-Patterns

- Running webhook delivery assertions without a live Svix instance — tests fail if SVIX_HOST is unset and assertions hit Svix API.
- Creating a second TestEnv inside a test method — each TestEnv starts a goroutine and opens DB connections; always reuse the suite-level env.
- Forgetting defer env.Close() — leaves the eventHandler goroutine and DB/Ent connections open.
- Using package-level time vars (TestEntitlementCurrentUsagePeriod) as exact assertion values in time-sensitive tests — value is computed at package init.
- Calling env.Notification() or env.NotificationRepo() before NewTestEnv completes — testEnv fields are only set after NewTestEnv returns.

## Decisions

- **TestEnv interface rather than exposing service structs directly to test suites.** — Decouples suites from concrete wiring in testenv.go; allows swapping Svix for a mock without modifying every test file.
- **Wire a real Svix client (not a mock) in testenv.go.** — Notification correctness depends on Svix application/endpoint/message creation semantics that a mock cannot catch; gated by SVIX_HOST availability.

## Example: Add a new balance-consumer test step to BalanceNotificaiontHandlerTestSuite

```
func (s *BalanceNotificaiontHandlerTestSuite) TestMyFlow(ctx context.Context, t *testing.T) {
	s.setupNamespace(ctx, t)
	service := s.Env.Notification()

	snapshotEvent := NewBalanceSnapshotEvent(BalanceSnapshotEventInput{
		Feature: s.feature,
		Value:   snapshot.EntitlementValue{Balance: convert.ToPointer(50.0)},
		Namespace: s.namespace,
	})
	require.NoError(t, s.handler.Handle(ctx, snapshotEvent))

	events, err := service.ListEvents(ctx, notification.ListEventsInput{Namespaces: []string{s.namespace}})
	require.NoError(t, err)
	require.Empty(t, events.Items)
}
```

<!-- archie:ai-end -->
