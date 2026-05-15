# server

<!-- archie:ai-start -->

> Binary entrypoint for the main HTTP API server: wires all ~40 domain services via Wire, runs DB migrations, registers namespace handlers in strict order (Ledger then KafkaIngest before initNamespace), provisions sandbox app and default billing profile, constructs router.Config with all services, and runs an oklog/run group with API server, telemetry server, Kafka producer, notification event handler, and termination checker.

## Patterns

**Namespace handler registration before initNamespace** — LedgerNamespaceHandler and KafkaIngestNamespaceHandler MUST be registered on NamespaceManager before initNamespace is called — otherwise the default namespace is provisioned without those handlers. (`app.NamespaceManager.RegisterHandler(app.LedgerNamespaceHandler)
app.NamespaceManager.RegisterHandler(app.KafkaIngestNamespaceHandler)
initNamespace(app.NamespaceManager, logger)`)
**router.Config aggregates all domain services** — server.NewServer receives a router.Config struct with every domain service as a named field. Adding a new endpoint requires adding the service to router.Config and wiring it in Application/wire.go. (`router.Config{ Billing: app.BillingRegistry.Billing, Customer: app.Customer, ChargeService: app.BillingRegistry.ChargesServiceOrNil(), ... }`)
**Run group with five components** — main.go assembles a run.Group with: telemetry server, Kafka producer (kafkaingest.KafkaProducerGroup), API HTTP server, notification event handler, termination checker, and signal handler — all with graceful shutdown. (`group.Add(kafkaingest.KafkaProducerGroup(ctx, app.KafkaProducer, logger, app.KafkaMetrics))
group.Add(apiServerRun, apiServerShutdown)
group.Add(eventHandlerStart, eventHandleStop)`)
**Post-migration provisioning before server start** — After Migrate: register namespace handlers → initNamespace → SandboxProvisioner → ProvisionDefaultBillingProfile → MeterConfigInitializer — all must succeed before server.NewServer and run.Group start. (`app.BillingRegistry.Billing.ProvisionDefaultBillingProfile(ctx, namespace)
app.MeterConfigInitializer(ctx)`)
**Use common.WatermillNoPublisher (not common.Watermill)** — cmd/server publishes Kafka events but does not consume them. Use common.WatermillNoPublisher in wire.Build — common.Watermill sets up a bidirectional subscriber+publisher which is wrong for the server binary. (`common.WatermillNoPublisher, // in wire.Build — server publishes via kafkaingest but does not subscribe`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Orchestrates full startup: config parse → Wire init → SetGlobals → Migrate → namespace handler registration → initNamespace → provisioning → server construction → run group. | Any new namespace.Handler must be registered BEFORE initNamespace(). Any new post-migration provisioning step must happen before run.Group.Run(). |
| `wire.go` | Largest Wire file: lists ~30 provider sets including common.App, common.Billing, common.LedgerStack, common.Kafka, common.KafkaIngest, common.Server. Add new service dependencies here. | common.LedgerStack is the credits-aware ledger provider — don't construct ledger services outside it. common.WatermillNoPublisher (not common.Watermill) — server publishes but does not consume Kafka events. |
| `wire_gen.go` | Generated — DO NOT EDIT. The longest generated file (~600 lines). Reference to understand dependency order. | credits-guarded ledger construction appears here — verify noop vs real implementation when debugging credits.enabled=false paths. |

## Anti-Patterns

- Registering a namespace.Handler after initNamespace — the default namespace will miss that handler's provisioning
- Adding domain service calls directly in main.go startup sequence without error handling and os.Exit(1)
- Manually editing wire_gen.go
- Adding a new HTTP endpoint without adding its service to router.Config and declaring it in wire.go Application struct
- Using common.Watermill instead of common.WatermillNoPublisher — server does not consume Kafka topics

## Decisions

- **cmd/server is the only binary that registers namespace handlers; other workers don't register Kafka ingest or ledger namespace handlers.** — Namespace provisioning (ClickHouse table creation, Kafka topic creation, ledger account setup) must happen exactly once via the server startup path.
- **NotificationEventHandler is run as a goroutine in cmd/server's run.Group, not in a separate binary.** — The notification event handler is lightweight and co-located with the API server to simplify deployment; a dedicated notification-service binary is also available for horizontal scaling.

## Example: Adding a new domain service to cmd/server and wiring it to an HTTP handler

```
// 1. Add to Application in wire.go:
MyService myservice.Service
// 2. Add provider set to wire.Build:
common.MyService,
// 3. Add to router.Config in main.go:
MyService: app.MyService,
// 4. Run: make generate
```

<!-- archie:ai-end -->
