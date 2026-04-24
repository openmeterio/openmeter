# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing app.Adapter — persists installed apps, app-customer relationships, and routes marketplace registry lookups. The in-memory registry map[AppType]RegistryItem is the only non-Ent state; all mutations go through TransactingRepo.

## Patterns

**TransactingRepo wrapping** — Every multi-step write (CreateApp, UpdateApp, UninstallApp, EnsureCustomer) is wrapped in transaction.Run + entutils.TransactingRepo so the ctx-bound Ent transaction is honoured. Read-only queries (ListApps, GetApp) use TransactingRepo without a prior transaction.Run. (`return transaction.Run(ctx, a, func(ctx context.Context) (app.App, error) { return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (app.App, error) { ... }) })`)
**WithTx + Self pattern** — adapter implements entutils.TxCreator via Tx(), WithTx(), and Self() so entutils.TransactingRepo can rebind the adapter to any transaction driver found in ctx. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), registry: a.registry} }`)
**Registry held on adapter struct** — The marketplace registry is an in-memory map[app.AppType]RegistryItem on the adapter. RegisterMarketplaceListing mutates it directly (value receiver, so pointer receiver must be used in practice — note the adapter's pointer is shared). GetMarketplaceListing returns models.GenericNotFoundError when the type is absent. (`a.registry[input.Listing.Type] = input`)
**Soft-delete via DeletedAt** — Apps and app-customer rows are never hard-deleted. UninstallApp sets DeletedAt on the App row; DeleteCustomer sets DeletedAt on AppCustomer. ListApps filters out deleted rows by default unless IncludeDeleted is true. (`query = query.Where(appdb.DeletedAtIsNil())`)
**mapAppFromDB delegates to factory** — mapAppFromDB calls registryItem.Factory.NewApp(ctx, appBase) to produce the concrete app.App; the adapter never type-switches on AppType itself for construction. (`app, err := registryItem.Factory.NewApp(ctx, appBase)`)
**EnsureCustomer upsert pattern** — AppCustomer rows are upserted with OnConflictColumns targeting (namespace, app_id, customer_id) and UpdateDeletedAt to restore soft-deleted rows. Workaround for ent issue #1821 where DoNothing() can return 'sql: no rows in result set'. (`OnConflictColumns(appcustomerdb.FieldNamespace, appcustomerdb.FieldAppID, appcustomerdb.FieldCustomerID).UpdateDeletedAt().Exec(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Struct definition, Config/New constructor, Tx/WithTx/Self for TransactingRepo, compile-time interface assertion var _ app.Adapter = (*adapter)(nil). | registry map is value-initialised to empty map in New(); adding new registry access must go through pointer receiver or the map is shared correctly via pointer. |
| `app.go` | CreateApp, GetApp, ListApps, UpdateApp, UninstallApp, UpdateAppStatus, and the two private mappers (mapAppBaseFromDB, mapAppFromDB). | UpdateAppStatus does NOT use TransactingRepo — it updates directly on a.db, bypassing any active ctx transaction. This is intentional (status is idempotent) but notable. |
| `customer.go` | ListCustomerData (delegates to ListApps + per-app GetCustomerData), EnsureCustomer (upsert), DeleteCustomer (soft-delete by appID and/or customerID). | ListCustomerData calls GetCustomerData on each App instance — this may trigger further Ent queries per app type. Namespace is derived from AppID or CustomerID; both nil => validation error. |
| `marketplace.go` | In-memory registry CRUD: RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing. OAuth2 methods return 'not implemented'. | RegisterMarketplaceListing panics on duplicate key (returns error but callers at startup must handle it). InstallMarketplaceListing type-asserts Factory to AppFactoryInstall — missing capability returns GenericValidationError. |

## Anti-Patterns

- Calling a.db directly inside a method that may already be inside a transaction — always go through entutils.TransactingRepo to rebind to the ctx-bound tx.
- Importing app-type-specific packages (appstripe, appsandbox) in this adapter — construction is delegated to the factory, keeping the adapter type-agnostic.
- Hard-deleting app or app-customer rows — the pattern is soft-delete via DeletedAt.
- Adding business logic to this adapter beyond persistence — service-layer orchestration belongs in openmeter/app/service.

## Decisions

- **Registry lives on the adapter (not a separate service layer)** — The marketplace registry must be consulted on every DB read (mapAppFromDB needs the factory) so co-locating it with the Ent client avoids a second dependency injection and keeps ListApps/GetApp atomic within one struct.
- **OAuth2 methods return 'not implemented'** — OAuth2 app installation is not yet supported; stubbing here keeps the adapter interface complete without forcing callers to handle nil adapters.

## Example: Creating a new app inside a transaction

```
func (a *adapter) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (app.AppBase, error) {
		return entutils.TransactingRepo(
			ctx, a,
			func(ctx context.Context, repo *adapter) (app.AppBase, error) {
				dbApp, err := repo.db.App.Create().
					SetNamespace(input.Namespace).
					SetName(input.Name).
					SetType(input.Type).
					SetStatus(app.AppStatusReady).
					Save(ctx)
				if err != nil { return app.AppBase{}, err }
				registryItem, _ := repo.GetMarketplaceListing(ctx, app.MarketplaceGetInput{Type: dbApp.Type})
				return mapAppBaseFromDB(dbApp, registryItem), nil
			})
// ...
```

<!-- archie:ai-end -->
