# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL persistence adapter for the custominvoicing app — implements appcustominvoicing.Adapter by reading and writing AppCustomInvoicing and AppCustomInvoicingCustomer Ent entities with soft-delete semantics and idempotent upserts.

## Patterns

**TransactingRepo wrapping on every method** — Every exported method body wraps Ent access in entutils.TransactingRepo or TransactingRepoWithNoValue so it rebinds to the caller's ctx-bound transaction. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { return tx.db.AppCustomInvoicing.Update().Where(...).SetDeletedAt(clock.Now()).Exec(ctx) })`)
**Soft-delete everywhere** — Records are never hard-deleted. Mutations set SetDeletedAt(clock.Now()); all queries add DeletedAtIsNil() predicate. (`tx.db.AppCustomInvoicing.Update().Where(appcustominvoicing.ID(id), appcustominvoicing.DeletedAtIsNil()).SetDeletedAt(clock.Now()).Exec(ctx)`)
**Upsert via OnConflictColumns** — Create operations use .OnConflictColumns(...).UpdateNewValues() for idempotent upserts without a read-then-write. (`.OnConflictColumns(appcustominvoicing.FieldID, appcustominvoicing.FieldNamespace).UpdateNewValues().Exec(ctx)`)
**Tx/WithTx/Self triad** — adapter implements Tx() (HijackTx + NewTxDriver), WithTx() (NewTxClientFromRawConfig), and Self() to enable entutils.TransactingRepo transaction rebinding. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**Compile-time interface assertion** — var _ appcustominvoicing.Adapter = (*adapter)(nil) at package level ensures the adapter satisfies the interface at compile time. (`var _ appcustominvoicing.Adapter = (*adapter)(nil)`)
**Config struct with Validate() constructor** — Constructor accepts a Config struct; Validate() checks all required fields before constructing the adapter. (`func New(config Config) (appcustominvoicing.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Zero-value return on NotFound** — GetAppConfiguration and GetCustomerData return an empty struct (not an error) when the Ent query returns IsNotFound — callers rely on this zero-value sentinel. (`if db.IsNotFound(err) { return custominvoicing.Configuration{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter constructor, Config, and transaction plumbing (Tx/WithTx/Self). Foundation for all other files. | Do not bypass WithTx/Self — other files rely on TransactingRepo rebinding through these methods. |
| `appconfig.go` | CRUD for app-level configuration (AppCustomInvoicing entity): GetAppConfiguration, UpsertAppConfiguration, DeleteAppConfiguration. | GetAppConfiguration returns empty Configuration{} (not an error) on NotFound — callers rely on this zero-value sentinel. |
| `customerdata.go` | CRUD for per-customer data (AppCustomInvoicingCustomer entity): GetCustomerData, UpsertCustomerData, DeleteCustomerData. | input.Validate() is called before entering the transaction — validation errors must not be wrapped in a transaction. |

## Anti-Patterns

- Using a.db directly in a method body without TransactingRepo — falls off ctx-bound transactions
- Hard-deleting rows instead of setting DeletedAt
- Read-then-write instead of OnConflictColumns upsert
- Returning an error on NotFound from GetAppConfiguration or GetCustomerData — callers expect zero-value returns
- Storing *entdb.Tx as a struct field instead of using the Tx/WithTx/Self triad

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
