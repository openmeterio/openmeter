# billing-worker

<!-- archie:ai-start -->

> Binary entrypoint for the billing-worker: subscribes to Kafka system events via Watermill, processes subscription sync, invoice creation, and charge advancement events. Performs post-migration provisioning (ledger business accounts, sandbox app) before starting the run loop.

## Patterns

**Post-migration provisioning before Run** — billing-worker calls LedgerAccountResolver.EnsureBusinessAccounts and AppRegistry.SandboxProvisioner after Migrate and before Run — ordering is critical for correct startup state. (`app.Migrate(ctx)
app.LedgerAccountResolver.EnsureBusinessAccounts(ctx, namespace)
app.AppRegistry.SandboxProvisioner(ctx, namespace)
app.Run()`)
**Extended Application struct for post-Migrate provisioning** — Application embeds common.GlobalInitializer, Migrator, Runner and additionally exposes AppRegistry, LedgerAccountResolver, NamespaceManager, Streaming as fields needed for post-migration provisioning in main.go. (`type Application struct { common.GlobalInitializer; common.Migrator; common.Runner; AppRegistry common.AppRegistry; LedgerAccountResolver ledger.AccountResolver; NamespaceManager *namespace.Manager; ... }`)
**BillingWorker composite provider set** — wire.Build uses common.BillingWorker which internally wires billing adapter, rating service, subscription sync, charges, and ledger stack. Individual billing sub-services are not listed separately. (`common.BillingWorker // in wire.Build — provisions billingRegistry, subscriptionsyncService, workerOptions, worker in sequence`)
**Config FieldsOf extraction for worker sub-config** — wire.FieldsOf extracts the router/consumer config slices for Wire from BillingWorkerConfiguration and BillingConfiguration. (`wire.FieldsOf(new(config.BillingWorkerConfiguration), "ConsumerConfiguration"), wire.FieldsOf(new(config.BillingConfiguration), "Worker")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Startup sequence: parse config → initializeApplication → SetGlobals → Migrate → EnsureBusinessAccounts → SandboxProvisioner → Run. | EnsureBusinessAccounts and SandboxProvisioner must run after Migrate but before Run. Adding new post-migration provisioning must respect this ordering. |
| `wire.go` | Declares Application struct and all provider sets. Edit here to add new service dependencies. | common.BillingWorker is a composite set — don't duplicate the billing sub-services it already includes. |
| `wire_gen.go` | Generated — DO NOT EDIT. Shows full wiring sequence including ledger stack, app registry, Stripe/CustomInvoicing/Sandbox apps. | Ledger stack wiring (NewLedgerHistoricalRepo, NewLedgerAccountService, etc.) is guarded by creditsConfiguration — verify credits path when debugging ledger writes. |

## Anti-Patterns

- Adding business logic to main.go — it belongs in openmeter/billing/worker or openmeter/billing/worker/subscriptionsync
- Calling EnsureBusinessAccounts or SandboxProvisioner after Run() — they must execute before the worker starts consuming events
- Manually editing wire_gen.go
- Duplicating billing sub-service provider sets already included by common.BillingWorker
- Bypassing the credits.enabled guard by directly constructing ledger services outside common.LedgerStack

## Decisions

- **Ledger business accounts and sandbox app are provisioned in billing-worker to support standalone deployments.** — When billing-worker runs independently of cmd/server, it must self-provision its required DB state before consuming Kafka events.
- **AppRegistry (Sandbox, Stripe, CustomInvoicing) is wired in billing-worker because invoice lifecycle events require app implementations registered before the Watermill router starts.** — Invoice state machine dispatches to the registered InvoicingApp; wiring must happen before event consumption begins.

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
