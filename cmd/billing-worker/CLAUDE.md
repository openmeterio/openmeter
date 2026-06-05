# billing-worker

<!-- archie:ai-start -->

> main.go entrypoint for the billing-worker binary (invoice advancement/collection + subscription->billing reconciliation). Beyond the shared bootstrap it also provisions ledger business accounts and the sandbox app on startup before Run().

## Patterns

**Extra startup provisioning steps** — After Migrate, main() calls app.LedgerAccountResolver.EnsureBusinessAccounts(ctx, app.NamespaceManager.GetDefaultNamespace()) and app.AppRegistry.SandboxProvisioner(...) before app.Run(); each error -> os.Exit(1). (`_, err = app.LedgerAccountResolver.EnsureBusinessAccounts(ctx, app.NamespaceManager.GetDefaultNamespace())`)
**Richer Application struct** — Application exposes AppRegistry, LedgerAccountResolver, Meter, NamespaceManager, Streaming alongside the GlobalInitializer/Migrator/Runner mixins so startup steps can reach those services. (`AppRegistry common.AppRegistry; LedgerAccountResolver ledger.AccountResolver; NamespaceManager *namespace.Manager`)
**Feature-gate noop in wire** — wire.Build includes common.FeatureGateNoopSet; the generated injector builds gate := featuregate.NewNoop() and threads it through NewBillingRegistry and NewBillingSubscriptionSyncService. (`common.FeatureGateNoopSet`)
**Worker config via FieldsOf** — wire.FieldsOf(new(config.BillingConfiguration), "Worker") and FieldsOf(...BillingWorkerConfiguration, "ConsumerConfiguration") extract nested config sub-structs for providers. (`wire.FieldsOf(new(config.BillingConfiguration), "Worker")`)
**metadata() names the binary** — metadata(conf) = common.NewMetadata(conf, version, "billing-worker"). (`common.NewMetadata(conf, version, "billing-worker")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Bootstrap + Migrate + ledger/sandbox provisioning + Run. | EnsureBusinessAccounts and SandboxProvisioner must stay before Run(); they depend on Migrate having run first. |
| `wire.go` | billing-worker provider list (common.BillingWorker, common.Streaming, common.TaxCode, common.FeatureGateNoopSet, ...). | credits.enabled gating is handled inside the wired ledger providers; do not bypass them here. |
| `wire_gen.go` | Generated injector wiring the large billing/ledger/subscription graph; DO NOT EDIT. | The injector constructs LedgerHistorical/Account/Resolver chains and BillingRegistry; regenerate via make generate. |
| `version.go` | ldflags version metadata. | Identical to other binaries. |

## Anti-Patterns

- Editing wire_gen.go instead of wire.go
- Running app.Run() before EnsureBusinessAccounts/SandboxProvisioner provisioning
- Hardcoding a real feature gate here instead of the wired FeatureGateNoopSet
- Adding billing logic to main.go rather than app/common providers

## Decisions

- **Startup eagerly provisions ledger business accounts and sandbox app for the default namespace** — The worker must have its accounts and sandbox integration ready before it begins reconciling/invoicing.

## Example: Post-migrate provisioning before Run()

```
if err := app.Migrate(ctx); err != nil { os.Exit(1) }
_, err = app.LedgerAccountResolver.EnsureBusinessAccounts(ctx, app.NamespaceManager.GetDefaultNamespace())
if err != nil { logger.Error("failed to provision ledger business accounts", "error", err); os.Exit(1) }
err = app.AppRegistry.SandboxProvisioner(ctx, app.NamespaceManager.GetDefaultNamespace())
if err != nil { os.Exit(1) }
app.Run()
```

<!-- archie:ai-end -->
