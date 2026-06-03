# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing app.Adapter — persists installed apps and app-customer relationships, and holds the in-memory marketplace registry map[AppType]RegistryItem that every DB read (mapAppFromDB) consults to construct concrete app.App instances via the registered factory.

## Patterns

**TransactingRepo on every mutating method** — State-changing methods (CreateApp, UpdateApp, UninstallApp, EnsureCustomer) wrap Ent calls in transaction.Run + entutils.TransactingRepo. Read-only methods (ListApps, GetApp) use TransactingRepo alone. (`return transaction.Run(ctx, a, func(ctx context.Context) (app.AppBase, error) { return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (app.AppBase, error) { ... }) })`)
**TxCreator + TxUser triad** — adapter implements Tx() via HijackTx+NewTxDriver, WithTx() via NewTxClientFromRawConfig, and Self(). All three are required for TransactingRepo to rebind to a ctx-carried transaction. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), registry: a.registry} }`)
**mapAppFromDB delegates construction to factory** — mapAppFromDB never type-switches on AppType; it calls registryItem.Factory.NewApp(ctx, appBase) to produce the concrete app.App, keeping the adapter type-agnostic. (`app, err := registryItem.Factory.NewApp(ctx, appBase)`)
**Soft-delete via DeletedAt** — Apps and app-customer rows are never hard-deleted. UninstallApp sets DeletedAt on App; DeleteCustomer sets DeletedAt on AppCustomer. ListApps filters deleted unless IncludeDeleted. (`query = query.Where(appdb.DeletedAtIsNil())`)
**EnsureCustomer upsert with OnConflictColumns** — AppCustomer rows are upserted on (namespace, app_id, customer_id); UpdateDeletedAt restores soft-deleted rows. The 'sql: no rows in result set' string is the known Ent issue #1821 and is silently ignored. (`OnConflictColumns(appcustomerdb.FieldNamespace, appcustomerdb.FieldAppID, appcustomerdb.FieldCustomerID).UpdateDeletedAt().Exec(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Struct, Config/New, Tx/WithTx/Self, compile-time assertion var _ app.Adapter = (*adapter)(nil). | registry map is value-initialised in New(); pointer receiver is essential so the registry is shared across all TransactingRepo-derived adapter copies. |
| `app.go` | CreateApp, GetApp, ListApps, UpdateApp, UninstallApp, UpdateAppStatus, plus mappers mapAppBaseFromDB and mapAppFromDB. | UpdateAppStatus uses TransactingRepo directly (no transaction.Run) — intentionally idempotent and not wrapped in a top-level transaction. |
| `customer.go` | ListCustomerData (ListApps + per-app GetCustomerData), EnsureCustomer (upsert), DeleteCustomer (soft-delete). | ListCustomerData calls GetCustomerData on each App which may trigger per-app-type Ent queries. Namespace derived from AppID or CustomerID; both nil => validation error. |
| `marketplace.go` | In-memory registry CRUD: RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing. | RegisterMarketplaceListing returns GenericConflictError on duplicate key. InstallMarketplaceListing type-asserts Factory to AppFactoryInstall (missing => GenericValidationError). OAuth2 methods return GenericNotImplementedError. |

## Anti-Patterns

- Calling a.db directly in a method body that may run inside a caller-supplied transaction — always use entutils.TransactingRepo to rebind
- Importing app-type-specific packages (appstripe, appsandbox, appcustominvoicing) here — construction is delegated to factory
- Hard-deleting app or app-customer rows — the pattern is soft-delete via DeletedAt
- Adding business logic beyond persistence — service-layer orchestration belongs in openmeter/app/service

## Decisions

- **In-memory registry lives on the adapter struct (not a separate layer)** — mapAppFromDB must consult the factory registry on every DB read; co-locating it with the Ent client keeps ListApps/GetApp atomic within one struct and avoids a second DI dependency.
- **OAuth2 methods return GenericNotImplementedError stubs** — OAuth2 app installation is not yet supported; stubbing keeps the adapter interface complete without forcing callers to handle nil adapters.

## Example: Creating a new app inside a transaction with registry lookup

```
func (a *adapter) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (app.AppBase, error) {
		return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (app.AppBase, error) {
			dbApp, err := repo.db.App.Create().SetNamespace(input.Namespace).SetName(input.Name).SetType(input.Type).SetStatus(app.AppStatusReady).Save(ctx)
			if err != nil { return app.AppBase{}, err }
			registryItem, err := repo.GetMarketplaceListing(ctx, app.MarketplaceGetInput{Type: dbApp.Type})
			if err != nil { return app.AppBase{}, err }
			return mapAppBaseFromDB(dbApp, registryItem), nil
		})
	})
}
```

<!-- archie:ai-end -->
