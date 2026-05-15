# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing charges.Adapter and charges.ChargesSearchAdapter; provides all DB access for the charges domain (search, advancement candidates) with strict transaction rebinding via entutils.TransactingRepo on every exported method.

## Patterns

**TransactingRepo on every method** — Every exported adapter method body must be wrapped in entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter) ...) so the method rebinds to the ctx-carried Ent transaction instead of the raw client. Never call tx.db.Foo() directly outside this wrapper. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (T, error) { return tx.db.ChargesSearchV1.Query()... })`)
**Tx / WithTx / Self triad** — adapter implements TxCreator via Tx() (HijackTx + NewTxDriver), TxUser[T] via WithTx() (NewTxClientFromRawConfig), and Self() — all three are required for TransactingRepo to correctly rebind or self-start transactions. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**Config.Validate() before construction** — New(Config) calls config.Validate() and returns an error if Client or Logger is nil before constructing the adapter struct. Never construct adapter directly. (`func New(config Config) (charges.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger}, nil }`)
**Interface compliance assertions** — Each interface the adapter implements has a compile-time var _ assertion at file scope. (`var _ charges.Adapter = (*adapter)(nil)
var _ charges.ChargesSearchAdapter = (*adapter)(nil)`)
**Manual Go-side pagination for GroupBy queries** — Ent's Paginate() is incompatible with GroupBy; ListCustomersToAdvance fetches all grouped results via Scan() then slices manually in Go for pagination. (`err := query.GroupBy(fields...).Scan(ctx, &results); start := page.Offset(); pageResults := results[start:end]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, and the adapter struct with Tx/WithTx/Self for transaction lifecycle. Shared by all query files in this package. | Never pass a.db directly to a helper — always go through tx *adapter inside TransactingRepo so the Ent client is bound to the current transaction. |
| `search.go` | Implements GetByIDs, ListCharges, and ListCustomersToAdvance using ChargesSearchV1 Ent view. Shows namespace-filter pattern and the GroupBy+manual-pagination workaround. | ChargesSearchV1 is a read-only Ent view — never attempt writes through it. GroupBy results cannot use query.Paginate(); paginate manually in Go. |
| `search_test.go` | Integration tests running against a real Postgres instance using testutils.InitPostgresDB and migrate.Up(). Each test uses isolated namespaces. | Tests use context.Background() at DB setup level; for new test methods prefer t.Context() where *testing.T is available. |

## Anti-Patterns

- Adding a method that reads or writes via a.db without wrapping in entutils.TransactingRepo — breaks ctx-carried transaction isolation
- Writing to ChargesSearchV1 — it is a read-only Ent view backed by underlying charge tables
- Using query.Paginate() after GroupBy — Ent does not support this combination; paginate manually in Go
- Constructing the adapter without calling config.Validate() — nil client causes a panic at query time
- Adding business logic (orchestration, state machine calls) to adapter methods — adapters must be pure data-access

## Decisions

- **entutils.TransactingRepo wraps every method body rather than wrapping at the service call site** — Charge advancement mixes reads and writes across multiple adapter calls in a single transaction; each helper must independently rebind to the ctx-carried tx to avoid partial writes.
- **Manual Go-side pagination for ListCustomersToAdvance instead of Ent Paginate** — Ent's generated Paginate() is incompatible with GROUP BY queries; total count is computed from the full scan and sliced in Go.

## Example: Adding a new query method that must respect ctx-carried transactions

```
func (a *adapter) GetActiveByCustomer(ctx context.Context, input charges.GetActiveInput) ([]charges.ChargeSearchItem, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]charges.ChargeSearchItem, error) {
		rows, err := tx.db.ChargesSearchV1.Query().
			Where(dbchargessearchv1.Namespace(input.Namespace)).
			Where(dbchargessearchv1.CustomerID(input.CustomerID)).
			Where(dbchargessearchv1.DeletedAtIsNil()).
			All(ctx)
		if err != nil {
			return nil, err
		}
		return lo.Map(rows, func(r *db.ChargesSearchV1, _ int) charges.ChargeSearchItem {
			return mapChargeSearchToChargeWithType(r)
		}), nil
	})
}
```

<!-- archie:ai-end -->
