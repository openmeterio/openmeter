# notification

<!-- archie:ai-start -->

> Integration test suite (package `notification`) for the notification domain: channels, rules, events, repository filtering, the balance-threshold consumer, and Svix webhook delivery. Uses an interface-based TestEnv backed by real Postgres plus a real (local) Svix server.

## Patterns

**Interface TestEnv + single TestNotification dispatcher** — TestEnv exposes NotificationRepo(), Notification(), NotificationWebhook(), Feature(), Meter(), Namespace(), Close(). TestNotification(t) builds one env via NewTestEnv(t, ctx, namespace), then runs Webhook/Channel/Rule/Event/Repository/Consumer subtests, each backed by a small XxxTestSuite{Env}. (`env, err := NewTestEnv(t, ctx, namespace); testSuite := ChannelTestSuite{Env: env}; t.Run("Create", func(t *testing.T){ testSuite.TestCreate(ctx, t) })`)
**Namespace from NewTestNamespace (ULID)** — namespace := NewTestNamespace(t) (alias of NewTestULID) is created once and passed into NewTestEnv; suites read it via s.Env.Namespace(). (`namespace := NewTestNamespace(t); env, _ := NewTestEnv(t, ctx, namespace)`)
**Real Svix + Postgres, mock meter/eventbus** — NewTestEnv connects to a Svix server (webhooksvix.New with svix.New using a JWT from NewSvixAuthToken), real Postgres (testutils.InitPostgresDB + Schema.Create), a mock meter adapter (meter/mockadapter), and eventbus.NewMock. Svix host/secret come from SVIX_HOST / SVIX_JWT_SECRET env (defaults 127.0.0.1 / DUMMY_JWT_SECRET). (`svixAPIKey, _ := NewSvixAuthToken(svixJWTSigningSecret); webhook, _ := webhooksvix.New(webhooksvix.Config{SvixAPIClient: svixAPIClient, ...})`)
**Event handler runs as a background goroutine** — NewTestEnv builds eventhandler.New (with a pglockx lock client) and starts it via `go eventHandler.Start()`; closerFunc must eventHandler.Close() plus close ent/PG drivers. Consumer tests rely on this running loop. (`go func(){ _ = eventHandler.Start() }(); closerFunc := func() error { return errors.Join(eventHandler.Close(), entClient.Close(), ...) }`)
**Builder helpers for inputs** — Channel/rule/event inputs use builder funcs like NewCreateChannelInput(namespace, name) returning notification.CreateChannelInput with ChannelTypeWebhook config; repository tests tag events with notification.AnnotationEventFeatureID/Key and AnnotationEventSubjectID/Key for filtering. (`createIn := NewCreateChannelInput(s.Env.Namespace(), "NotificationCreateChannel"); channel, err := service.CreateChannel(ctx, createIn)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | TestEnv interface + NewTestEnv: wires notification adapter/service/webhook(Svix)/eventhandler, feature connector, mock meter; defines Test* constants (TestFeatureKey/ID, TestSubjectKey/ID, TestWebhookURL, TestSigningSecret) and NewTestNamespace | Requires a reachable Svix server (SvixServerURLTemplate http://host:8071) — these tests need make test-all / Svix dependency. Event handler goroutine + pglockx lock must be closed via closerFunc. |
| `helpers.go` | NewSvixAuthToken(signingSecret) (HS256 JWT for svix-server) and NewClickhouseClient(addr) | JWT issuer is hardcoded "svix-server" with a fixed expiry; ClickHouse creds are the local default/default. |
| `notification_test.go` | TestNotification entry point dispatching Webhook/Channel/Rule/Event/Repository/Consumer suites | Several suites call testSuite.Setup(ctx, t) before subtests; skipping Setup leaves rules/channels/events uncreated. |
| `repository.go` | RepositoryTestSuite: creates channel+rule+events with feature/subject annotations, tests ListEvents filtering by Features/Subjects | Feature filtering matches either the feature ID or key (TestFeatureID/TestFeatureKey); annotations are the AnnotationEvent* constants. |
| `channel.go` | ChannelTestSuite + NewCreateChannelInput builder (webhook channel with custom headers, URL, signing secret) | Channel config carries SigningSecret; webhook secret helpers live in notification/webhook/secret. |
| `consumer_balance.go` | BalanceNotificaiontHandlerTestSuite for the balance-threshold consumer (granting flow, feature filtering) | Depends on the running eventHandler goroutine to process events; assertions may need to wait for async delivery. |
| `webhook.go` | WebhookTestSuite exercising Svix webhook CRUD (create/update/delete/get/list) | Hits the live Svix server; flaky/skipped without the Svix dependency up. |

## Anti-Patterns

- Assuming no external dependency — these tests need a real Svix server (and Postgres); they are gated behind make test-all, not plain make test.
- Forgetting closerFunc/Close (eventHandler.Close + driver closes) — leaks the goroutine, lock, and DB.
- Hardcoding a namespace instead of NewTestNamespace(t); namespaces isolate the shared env.
- Skipping testSuite.Setup(ctx, t) before subtests that depend on pre-created channels/rules/events.
- Filtering events by raw map access instead of the notification.AnnotationEvent* constants.

## Decisions

- **TestEnv runs a real eventhandler goroutine plus a real Svix client** — Notification delivery and the balance-threshold consumer are inherently async and provider-backed; only a running handler + real Svix exercise the true delivery path.
- **Meter and eventbus are mocked while Postgres and Svix are real** — Channel/rule/event persistence and webhook delivery are the system under test; metering and event publishing are stubbed to keep scenarios deterministic.

## Example: Construct the env and run a channel subtest

```
func TestNotification(t *testing.T) {
  ctx, cancel := context.WithCancel(t.Context()); defer cancel()
  namespace := NewTestNamespace(t)
  env, err := NewTestEnv(t, ctx, namespace)
  require.NoError(t, err)
  t.Cleanup(func(){ _ = env.Close() })
  testSuite := ChannelTestSuite{Env: env}
  t.Run("Create", func(t *testing.T){ testSuite.TestCreate(ctx, t) })
}
```

<!-- archie:ai-end -->
