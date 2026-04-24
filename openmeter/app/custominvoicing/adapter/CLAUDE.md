# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL persistence adapter for the custominvoicing app domain — implements appcustominvoicing.Adapter by reading and writing AppCustomInvoicing and AppCustomInvoicingCustomer Ent entities. All mutations are soft-deletes (SetDeletedAt) and all queries filter DeletedAtIsNil().

## Patterns

**TransactingRepo wrapping** — Every method body is wrapped in entutils.TransactingRepo or entutils.TransactingRepoWithNoValue so it rebinds to any transaction already carried in ctx rather than executing on the raw client. (`entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { return tx.db.AppCustomInvoicing... })`)
**Soft-delete pattern** — Records are never hard-deleted. Mutations set SetDeletedAt(time.Now()); queries always add DeletedAtIsNil() predicate. (`tx.db.AppCustomInvoicing.Update().Where(appcustominvoicing.DeletedAtIsNil()).SetDeletedAt(time.Now()).Exec(ctx)`)
**Upsert via OnConflictColumns** — Create operations use OnConflictColumns(...).UpdateNewValues() to achieve idempotent upserts without a read-then-write. (`.OnConflictColumns(appcustominvoicing.FieldID, appcustominvoicing.FieldNamespace).UpdateNewValues().Exec(ctx)`)
**Tx/WithTx/Self triad for transaction plumbing** — adapter implements Tx() (HijackTx + NewTxDriver), WithTx() (NewTxClientFromRawConfig), and Self() so entutils.TransactingRepo can reuse the ctx-bound transaction. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**Compile-time interface assertion** — var _ appcustominvoicing.Adapter = (*adapter)(nil) at package level ensures the adapter satisfies the interface at compile time. (`var _ appcustominvoicing.Adapter = (*adapter)(nil)`)
**Config struct with Validate()** — Constructor accepts a Config struct with a Validate() method that checks all required fields before constructing the adapter. (`func New(config Config) (appcustominvoicing.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter constructor, Config, and transaction plumbing (Tx/WithTx/Self). Foundation for all other files in this package. | Do not bypass WithTx/Self — other files rely on TransactingRepo rebinding through these methods. |
| `appconfig.go` | CRUD for app-level configuration (AppCustomInvoicing entity): GetAppConfiguration, UpsertAppConfiguration, DeleteAppConfiguration. | GetAppConfiguration returns an empty Configuration{} (not an error) on NotFound — callers rely on this zero-value sentinel. |
| `customerdata.go` | CRUD for per-customer data (AppCustomInvoicingCustomer entity): GetCustomerData, UpsertCustomerData, DeleteCustomerData. | Input.Validate() is called at the top of each method before entering the transaction — validation errors should not be wrapped in a transaction. |

## Anti-Patterns

- Using a.db directly in a method body instead of going through TransactingRepo/TransactingRepoWithNoValue — falls off ctx-bound transactions
- Hard-deleting rows instead of setting DeletedAt
- Reading then writing instead of using OnConflictColumns upsert
- Returning an error on NotFound from GetAppConfiguration or GetCustomerData — callers expect zero-value returns

## Decisions

- **Soft-delete everywhere** — Preserves audit trail for billing apps; aligns with the rest of the billing domain's deletion convention.
- **Tx/WithTx/Self triad instead of passing *entdb.Tx explicitly** — Keeps transaction plumbing implicit in ctx; entutils.TransactingRepo handles rebinding without leaking *entdb.Tx into every call site.

## Example: Add a new adapter method that writes a record inside the caller's transaction

```
func (a *adapter) SetFlag(ctx context.Context, input appcustominvoicing.SetFlagInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.AppCustomInvoicing.Update().
			Where(appcustominvoicing.ID(input.ID), appcustominvoicing.Namespace(input.Namespace), appcustominvoicing.DeletedAtIsNil()).
			SetFlag(input.Value).
			Exec(ctx)
	})
}
```

<!-- archie:ai-end -->
