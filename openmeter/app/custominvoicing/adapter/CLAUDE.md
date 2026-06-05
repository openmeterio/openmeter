# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for the custom-invoicing app: stores per-app sync-hook configuration (AppCustomInvoicing) and per-customer external-system data (AppCustomInvoicingCustomer). Implements the appcustominvoicing.Adapter / AppConfigAdapter interfaces.

## Patterns

**Transaction-aware adapter with Tx/WithTx/Self** — adapter implements the entutils transacting contract: Tx hijacks an Ent tx, WithTx rebinds the client from raw config, Self returns itself. Every query method body is wrapped in entutils.TransactingRepo / TransactingRepoWithNoValue so it joins any tx already in ctx. (`func (a *adapter) GetAppConfiguration(ctx, input) (Configuration, error) { return entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter) (...){ tx.db.AppCustomInvoicing.Query()... }) }`)
**Config-validated constructor** — New(Config) validates Client and Logger are non-nil before returning the appcustominvoicing.Adapter interface; compile-time assert `var _ appcustominvoicing.Adapter = (*adapter)(nil)`. (`func New(config Config) (appcustominvoicing.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Soft delete via DeletedAt + IsNotFound returns empty** — Queries always filter DeletedAtIsNil(); deletes SetDeletedAt(time.Now()) instead of physical delete. Get-style methods translate db.IsNotFound into a zero-value struct + nil error rather than an error. (`if db.IsNotFound(err) { return custominvoicing.Configuration{}, nil }`)
**Upsert via OnConflict + Update partial columns** — Writes use Create().OnConflict... .UpdateNewValues() (config) or sql.ConflictColumns/ConflictWhere with UpdateMetadata()/UpdateDeletedAt() (customer data) so re-installs and metadata refreshes are idempotent. Customer-data conflict target is scoped by a partial unique index ConflictWhere(IsNull(FieldDeletedAt)). (`Create().OnConflict(sql.ConflictColumns(FieldCustomerID, FieldNamespace, FieldAppID), sql.ConflictWhere(sql.IsNull(FieldDeletedAt))).UpdateMetadata().UpdateDeletedAt().Exec(ctx)`)
**Input validation at adapter boundary** — Customer-data methods call input.Validate() before touching the DB; appconfig methods do not (callers validate upstream). (`if err := input.Validate(); err != nil { return appcustominvoicing.CustomerData{}, err }`)
**DB->domain mapping helpers** — Lowercase mapDBToAppConfiguration / mapDBToCustomerData translate the generated db.* row into the domain struct; never expose db.* types outward. (`func mapDBToAppConfiguration(appConfig *db.AppCustomInvoicing) custominvoicing.Configuration { return custominvoicing.Configuration{EnableDraftSyncHook: appConfig.EnableDraftSyncHook, ...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | adapter struct, Config/Validate, New constructor, Tx/WithTx/Self transaction plumbing | WithTx must rebuild via entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); don't reuse the outer a.db inside a tx |
| `appconfig.go` | Get/Upsert/Delete AppConfiguration (sync-hook flags) | Upsert uses OnConflictColumns(FieldID, FieldNamespace).UpdateNewValues(); keyed by app ID+namespace, not customer |
| `customerdata.go` | Get/Upsert/Delete CustomerData keyed by CustomerID+Namespace+AppID | input.Validate() required here; conflict target includes the partial-index ConflictWhere on DeletedAt |

## Anti-Patterns

- Running Ent queries outside entutils.TransactingRepo/TransactingRepoWithNoValue, breaking tx propagation from ctx
- Returning an error instead of a zero value + nil on db.IsNotFound
- Physical deletes instead of SetDeletedAt soft delete
- Returning generated db.* types instead of mapping to domain structs
- Forgetting DeletedAtIsNil() in query/update Where clauses

## Decisions

- **Two separate Ent tables (AppCustomInvoicing for app config, AppCustomInvoicingCustomer for per-customer data)** — App-level sync-hook config and customer-to-external-ID mapping have different keys and lifecycles
- **Get-on-missing returns empty struct, not NotFound error** — Sync-hook config and customer data are optional; callers treat absence as defaults

## Example: Upsert app sync-hook configuration idempotently

```
func (a *adapter) UpsertAppConfiguration(ctx context.Context, input custominvoicing.UpsertAppConfigurationInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.AppCustomInvoicing.Create().
			SetID(input.AppID.ID).
			SetNamespace(input.AppID.Namespace).
			SetEnableDraftSyncHook(input.Configuration.EnableDraftSyncHook).
			SetEnableIssuingSyncHook(input.Configuration.EnableIssuingSyncHook).
			OnConflictColumns(appcustominvoicing.FieldID, appcustominvoicing.FieldNamespace).
			UpdateNewValues().
			Exec(ctx)
	})
}
```

<!-- archie:ai-end -->
