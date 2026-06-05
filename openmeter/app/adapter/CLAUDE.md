# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for the app/marketplace framework. Implements app.Adapter (app CRUD, app-customer links) plus the in-memory marketplace registry (app.AppType -> app.RegistryItem) that drives app installation via per-type factories.

## Patterns

**Transaction-aware repo via entutils** — Write methods wrap bodies in transaction.Run/RunWithNoValue + entutils.TransactingRepo(ctx, a, func(ctx, repo *adapter)...) so they rebind to the tx client carried in ctx. The adapter implements Tx/WithTx/Self for entutils.TxCreator. (`func (a *adapter) CreateApp(...) { return transaction.Run(ctx, a, func(ctx) { return entutils.TransactingRepo(ctx, a, func(ctx, repo *adapter){ repo.db.App.Create()... }) }) }`)
**Registry is an in-memory map, not a table** — Marketplace listings live in adapter.registry map[app.AppType]app.RegistryItem populated by RegisterMarketplaceListing at wiring time; List/GetMarketplaceListing read from it, they never hit the DB. (`a.registry[input.Listing.Type] = input`)
**DB rows mapped through factory to live App** — GetApp/ListApps load *db.App, look up the RegistryItem by Type, then mapAppFromDB calls registryItem.Factory.NewApp(ctx, appBase) to produce the typed app.App. Never return raw db rows. (`mapAppFromDB(ctx, dbApp, registryItem) -> registryItem.Factory.NewApp(ctx, appBase)`)
**Soft delete via DeletedAt** — Apps and app-customer links are soft-deleted with SetDeletedAt(time.Now()); queries filter DeletedAtIsNil() unless IncludeDeleted. UninstallApp also calls Factory.UninstallApp before marking deleted. (`query = query.Where(appdb.DeletedAtIsNil())`)
**Upsert app-customer with conflict-on-not-deleted** — EnsureCustomer uses OnConflict over (namespace, app_id, customer_id) with ConflictWhere IsNull(deleted_at) to restore/upsert links; constraint errors are mapped to NewAppNotFoundError. (`OnConflict(sql.ConflictColumns(...), sql.ConflictWhere(sql.IsNull(appcustomerdb.FieldDeletedAt))).UpdateDeletedAt()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config{Client *entdb.Client} + New(); adapter struct holds db + registry; Tx/WithTx/Self implement entutils tx interface | WithTx must propagate the registry map to the new tx-bound adapter or marketplace lookups break inside transactions |
| `app.go` | App CRUD (CreateApp/GetApp/ListApps/UpdateApp/UninstallApp) + mapAppBaseFromDB/mapAppFromDB | GetApp returns NewAppNotFoundError on db.IsNotFound; UninstallApp must call Factory.UninstallApp before soft delete |
| `customer.go` | ListCustomerData/EnsureCustomer/DeleteCustomer for app-customer links | Ent issue #1821 workaround: treats 'sql: no rows in result set' from upsert DoNothing as success |
| `marketplace.go` | Registry-backed listing + install (InstallMarketplaceListing[WithAPIKey]); RegisterMarketplaceListing validates + rejects duplicates | Install branches on Factory satisfying app.AppFactoryInstall / AppFactoryInstallWithAPIKey via type assertion; Oauth2 methods return 'not implemented' |

## Anti-Patterns

- Calling repo.db.* directly inside a write method without wrapping in entutils.TransactingRepo/transaction.Run
- Persisting marketplace listings to the DB instead of the in-memory registry map
- Returning raw *db.App instead of routing through Factory.NewApp
- Hard-deleting apps/app-customer rows instead of setting DeletedAt

## Decisions

- **Marketplace registry is process-local in-memory state on the adapter** — App types are compiled-in plugins registered at startup; their factories cannot be serialized, so listings live in memory not Postgres
- **App identity split into db row + factory-built behavior** — DB stores generic app metadata; type-specific behavior (Stripe, sandbox, custom invoicing) is constructed lazily by the registered Factory

## Example: Transaction-aware create that maps a db row to a typed app via the registry

```
func (a *adapter) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (app.AppBase, error) {
		return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (app.AppBase, error) {
			dbApp, err := repo.db.App.Create().SetNamespace(input.Namespace).SetType(input.Type).SetStatus(app.AppStatusReady).Save(ctx)
			if err != nil { return app.AppBase{}, err }
			registryItem, err := repo.GetMarketplaceListing(ctx, app.MarketplaceGetInput{Type: dbApp.Type})
			if err != nil { return app.AppBase{}, err }
			return mapAppBaseFromDB(dbApp, registryItem), nil
		})
	})
}
```

<!-- archie:ai-end -->
