# billing-worker

<!-- archie:ai-start -->

> Binary entrypoint for the billing-worker: subscribes to Kafka system events via Watermill and processes subscription sync, invoice creation, and charge advancement events. Performs post-migration provisioning (ledger business accounts, sandbox app) before starting the run loop.

## Patterns

**Post-migration provisioning before Run** — main.go calls app.LedgerAccountResolver.EnsureBusinessAccounts and app.AppRegistry.SandboxProvisioner after Migrate and strictly before Run — ordering is critical for correct startup state. (`app.Migrate(ctx); app.LedgerAccountResolver.EnsureBusinessAccounts(ctx, app.NamespaceManager.GetDefaultNamespace()); app.AppRegistry.SandboxProvisioner(ctx, app.NamespaceManager.GetDefaultNamespace()); app.Run()`)
**Extended Application struct for post-Migrate provisioning** — Beyond the shared GlobalInitializer/Migrator/Runner embeds, Application exposes AppRegistry, LedgerAccountResolver, NamespaceManager, Streaming, and Meter as fields so main.go can drive provisioning. (`type Application struct { common.GlobalInitializer; common.Migrator; common.Runner; AppRegistry common.AppRegistry; LedgerAccountResolver ledger.AccountResolver; NamespaceManager *namespace.Manager; Streaming streaming.Connector; ... }`)
**BillingWorker composite provider set** — wire.Build uses common.BillingWorker which internally wires billing adapter, rating service, subscription sync, charges, and the ledger stack. Individual billing sub-services are not listed separately. (`common.BillingWorker // wires billingRegistry, subscriptionsyncService, workerOptions, worker in sequence`)
**Config FieldsOf extraction for worker sub-config** — wire.FieldsOf extracts the consumer/worker config slices for Wire from BillingWorkerConfiguration and BillingConfiguration. (`wire.FieldsOf(new(config.BillingWorkerConfiguration), "ConsumerConfiguration"), wire.FieldsOf(new(config.BillingConfiguration), "Worker")`)
**FeatureGateNoopSet wiring** — wire.Build includes common.FeatureGateNoopSet so featuregate.NewNoop() is injected; the billing-worker runs with a noop feature gate rather than a live gate provider. (`common.FeatureGateNoopSet // in wire.Build -> gate := featuregate.NewNoop()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Startup sequence: parse config -> initializeApplication -> SetGlobals -> Migrate -> EnsureBusinessAccounts -> SandboxProvisioner -> Run. | EnsureBusinessAccounts and SandboxProvisioner must run after Migrate but before Run. Any new post-migration provisioning must respect this ordering and use GetDefaultNamespace(). |
| `wire.go` | Declares the extended Application struct and all provider sets including common.BillingWorker, common.TaxCode, and common.FeatureGateNoopSet. | common.BillingWorker is a composite set — don't duplicate the billing sub-services it already includes. Has //go:build wireinject; never remove. |
| `wire_gen.go` | Generated — shows the full wiring sequence including ledger stack (NewLedgerHistoricalRepo, NewLedgerAccountService, ...), AppRegistry, and Stripe/CustomInvoicing/Sandbox apps. | DO NOT EDIT. The ledger stack wiring is gated by creditsConfiguration — verify the credits path when debugging ledger writes. |

## Anti-Patterns

- Adding business logic to main.go — it belongs in openmeter/billing/worker or openmeter/billing/worker/subscriptionsync
- Calling EnsureBusinessAccounts or SandboxProvisioner after Run() — they must execute before the worker starts consuming events
- Manually editing wire_gen.go
- Duplicating billing sub-service provider sets already included by common.BillingWorker
- Bypassing the credits.enabled guard by directly constructing ledger services outside common.LedgerStack

## Decisions

- **Ledger business accounts and the sandbox app are provisioned in billing-worker.** — When billing-worker runs independently of cmd/server it must self-provision its required DB state before consuming Kafka events.
- **AppRegistry (Sandbox, Stripe, CustomInvoicing) is wired in billing-worker.** — Invoice state-machine events dispatch to the registered InvoicingApp, so app implementations must be registered before the Watermill router starts consuming.

## Example: Correct post-Migrate startup sequence in billing-worker main.go

```
if err := app.Migrate(ctx); err != nil { logger.Error(...); os.Exit(1) }
_, err = app.LedgerAccountResolver.EnsureBusinessAccounts(ctx, app.NamespaceManager.GetDefaultNamespace())
if err != nil { logger.Error(...); os.Exit(1) }
err = app.AppRegistry.SandboxProvisioner(ctx, app.NamespaceManager.GetDefaultNamespace())
if err != nil { logger.Error(...); os.Exit(1) }
app.Run()
```

<!-- archie:ai-end -->
