# notification

<!-- archie:ai-start -->

> Integration tests for the notification domain: channels, rules, events, delivery status, webhook delivery via Svix, and balance-threshold consumer handler. Uses a dedicated TestEnv (testenv.go) that wires the full notification stack including Svix client from environment variables.

## Patterns

**TestEnv interface for service access** — All test suites receive a TestEnv value and call env.Notification(), env.NotificationRepo(), env.Feature(), env.Meter(), env.NotificationWebhook() instead of accessing service structs directly. TestEnv is created via NewTestEnv(t, ctx, namespace). (`env, err := NewTestEnv(t, ctx, namespace); service := env.Notification()`)
**Sub-suite pattern with Setup(ctx, t) and named test methods** — Test behaviour is split into XxxTestSuite structs (ChannelTestSuite, RuleTestSuite, EventTestSuite, RepositoryTestSuite, WebhookTestSuite) each with a Setup method. The top-level TestNotification function calls each suite's Setup then runs subtests by calling individual TestXxx methods. (`channelSuite := &ChannelTestSuite{Env: env}; channelSuite.Setup(ctx, t); t.Run("TestCreate", func(t *testing.T) { channelSuite.TestCreate(ctx, t) })`)
**Svix requires SVIX_HOST env var; falls back to DefaultSvixHost** — testenv.go reads SVIX_HOST (defaults to localhost) and SVIX_JWT_SECRET to connect to a real Svix instance. Webhook tests that assert delivery (not just API calls) require `docker compose up -d svix` or SVIX_HOST to be set. (`svixHost := defaultx.IfZero(os.Getenv("SVIX_HOST"), DefaultSvixHost)`)
**NewBalanceSnapshotEvent for balance consumer tests** — BalanceNotificaiontHandlerTestSuite uses NewBalanceSnapshotEvent(BalanceSnapshotEventInput{Feature, Value, Namespace}) to construct synthetic snapshot.SnapshotEvent values passed to handler.Handle(ctx, event). (`snapshotEvent := NewBalanceSnapshotEvent(BalanceSnapshotEventInput{Feature: s.feature, Value: snapshot.EntitlementValue{Balance: convert.ToPointer(50.0), ...}, Namespace: s.namespace})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Boots notification adapter (Ent), Svix webhook handler, notification event handler (goroutine), notification service, feature connector, and meter service. Returns closerFunc that shuts down eventHandler and closes DB connections. | eventHandler.Start() runs in a goroutine — tests must call env.Close() (which calls eventHandler.Close()) or use defer env.Close(). Svix connectivity is required for webhook delivery tests. |
| `consumer_balance.go` | BalanceNotificaiontHandlerTestSuite and BalanceSnapshotEventInput/NewBalanceSnapshotEvent helpers. Tests multi-step threshold crossing and reset deduplication logic of the balance consumer handler. | TestEntitlementCurrentUsagePeriod is a package-level var computed at init time (time.Now()). Tests that depend on exact period boundaries may be flaky if run near period boundaries. |
| `helpers.go` | Stateless helpers: NewSvixAuthToken (generates JWT for Svix API auth) and NewClickhouseClient (for notification tests that need ClickHouse). Not a test file — no *testing.T parameter. | NewClickhouseClient is present but notification tests do not currently use ClickHouse directly — it exists for potential future use. |
| `notification_test.go` | Top-level test entrypoint: creates TestEnv, constructs all sub-suite instances, calls Setup on each, and runs subtests. Single TestNotification function gates all sub-suite execution. | All sub-suites share the same TestEnv and namespace — sub-test data created in ChannelTestSuite may be visible to RuleTestSuite. Use distinct names/IDs to avoid collisions. |

## Anti-Patterns

- Running webhook delivery assertions without a live Svix instance — tests will fail if SVIX_HOST is unset and the test makes actual delivery assertions.
- Creating a second TestEnv inside a test method — each TestEnv starts a goroutine and opens DB connections; always reuse the suite-level env.
- Calling env.Notification() or env.NotificationRepo() before NewTestEnv completes — testEnv fields are set only after NewTestEnv returns.
- Using package-level time vars (TestEntitlementCurrentUsagePeriod) as exact assertion values in time-sensitive tests.
- Forgetting defer env.Close() — leaves the eventHandler goroutine and DB connections open.

## Decisions

- **Use a TestEnv interface rather than exposing service structs directly to test suites.** — Decouples test suites from the concrete wiring in testenv.go, allowing the wiring to change (e.g., swap Svix for a mock) without modifying every test suite.
- **Wire a real Svix client (not a mock) in testenv.go.** — Notification correctness depends on Svix application/endpoint/message creation semantics; a mock would not catch Svix API contract violations. The test is gated by SVIX_HOST availability.

<!-- archie:ai-end -->
