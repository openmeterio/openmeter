# server

<!-- archie:ai-start -->

> main.go entrypoint and DI bootstrap for the primary API server binary. Builds the full Application via Wire, migrates the DB, registers namespace handlers, provisions defaults, then serves the HTTP API + telemetry + notification event handler via an oklog/run group.

## Patterns

**Migrate then register handlers then initNamespace** — Order is fixed: app.Migrate(ctx) -> RegisterHandler(LedgerNamespaceHandler, KafkaIngestNamespaceHandler, TaxCodeNamespaceHandler) -> initNamespace(manager) -> SandboxProvisioner -> ProvisionDefaultBillingProfile -> MeterConfigInitializer. New handlers that must provision the default namespace MUST be registered before initNamespace. (`app.NamespaceManager.RegisterHandler(app.LedgerNamespaceHandler)`)
**Router config assembled from Application fields** — server.NewServer(&server.Config{RouterConfig: router.Config{...}}) maps ~40 app services (Billing, Customer, Plan, Subscription, Entitlement*, Charge via app.BillingRegistry.ChargesServiceOrNil(), FeatureGate, etc.) into the router. (`ChargeService: app.BillingRegistry.ChargesServiceOrNil()`)
**Debug connector wraps streaming** — debugConnector := debug.NewDebugConnector(app.StreamingConnector) is created and passed as RouterConfig.DebugConnector. (`debugConnector := debug.NewDebugConnector(app.StreamingConnector)`)
**Multi-actor run.Group** — group.Add for telemetry server, kafkaingest.KafkaProducerGroup, the API http.Server (timeouts from conf.Server), NotificationEventHandler, termination checker, and SignalHandler; group.Run(run.WithReverseShutdownOrder()). (`group.Add(kafkaingest.KafkaProducerGroup(ctx, app.KafkaProducer, logger, app.KafkaMetrics))`)
**Shared config bootstrap + panic funnel** — Same viper/pflag/DecodeHook load, conf.Validate(), defer log.PanicLogger(log.WithExit) as the workers. (`defer log.PanicLogger(log.WithExit)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Full server lifecycle: config, build app, migrate, register namespace handlers, provision defaults, mount router + /version, run actors. | Namespace handlers must be registered BEFORE initNamespace; SandboxProvisioner/ProvisionDefaultBillingProfile/MeterConfigInitializer run after the namespace exists. |
| `wire.go` | Largest provider list (billing, ledger, customer, productcatalog, portal, ingest, notification, secret, registry, taxcode, feature gate, ...). | Adding a router dependency means wiring its provider here AND adding the field to router.Config in main.go. |
| `wire_gen.go` | Generated full-application injector; DO NOT EDIT. | Regenerate via make generate; do not reorder generated cleanup chains. |
| `version.go` | ldflags version metadata; also served by the /version endpoint with runtime.GOOS/GOARCH. | Identical init() to other binaries. |

## Anti-Patterns

- Registering a namespace handler after initNamespace when it must provision the default namespace
- Editing wire_gen.go instead of wire.go
- Adding a router service without both a Wire provider and a router.Config field
- Calling provisioning steps before app.Migrate(ctx)
- Adding actors to the run.Group that break reverse-shutdown ordering

## Decisions

- **DB migration runs before default-namespace provisioning** — Namespace handlers and default-account/sandbox/billing-profile provisioning write rows that require the migrated schema.
- **All HTTP/telemetry/Kafka/notification actors run under one oklog/run group with reverse shutdown** — Coordinated graceful shutdown: dependents stop before their dependencies (e.g. API before Kafka producer).

## Example: Register namespace handlers before creating the default namespace

```
if err := app.Migrate(ctx); err != nil { os.Exit(1) }
if err = app.NamespaceManager.RegisterHandler(app.LedgerNamespaceHandler); err != nil { os.Exit(1) }
if err = app.NamespaceManager.RegisterHandler(app.KafkaIngestNamespaceHandler); err != nil { os.Exit(1) }
if err = initNamespace(app.NamespaceManager, logger); err != nil { os.Exit(1) }
```

<!-- archie:ai-end -->
