# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing charges.Adapter and charges.ChargesSearchAdapter. Provides all DB access for the charges domain — search queries, customer advancement candidates — with strict transaction rebinding via entutils.TransactingRepo on every method.

## Patterns

**TransactingRepo on every method** — Every exported adapter method body must be wrapped in entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter) ...) so the method rebinds to the ctx-carried transaction instead of the raw client. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (T, error) { return tx.db.Entity.Query()... })`)
**Tx / WithTx / Self transaction lifecycle** — adapter implements transaction.TxCreator via Tx() (HijackTx + NewTxDriver), WithTx() (NewTxClientFromRawConfig), and Self() — this triad is required for entutils.TransactingRepo to work correctly. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**Config.Validate() before construction** — New(Config) calls config.Validate() and returns an error if Client or Logger is nil before constructing the adapter struct. (`func New(config Config) (charges.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger}, nil }`)
**Interface compliance assertion** — Every interface the adapter implements has a compile-time assertion: var _ charges.ChargesSearchAdapter = (*adapter)(nil). (`var _ charges.Adapter = (*adapter)(nil)
var _ charges.ChargesSearchAdapter = (*adapter)(nil)`)
**Manual pagination for GroupBy queries** — Ent's Paginate() does not work with GroupBy; ListCustomersToAdvance applies Go-side slice pagination after fetching all grouped results with Scan(). (`err := query.GroupBy(fields...).Scan(ctx, &results); ... pageResults := results[start:end]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, and the adapter struct with Tx/WithTx/Self for transaction lifecycle. All query files in this package share this adapter struct. | Never pass a.db directly to a helper — always go through a tx *adapter inside TransactingRepo so the Ent client is bound to the current transaction. |
| `search.go` | Implements GetByIDs, ListCharges, and ListCustomersToAdvance using ChargesSearchV1 Ent view. Shows namespace-filter pattern and the GroupBy+manual-pagination workaround. | ChargesSearchV1 is a read-only Ent view — never attempt writes through it. GroupBy results cannot use query.Paginate(); slice manually. |
| `search_test.go` | Integration tests for the adapter that run against a real Postgres instance using testutils.InitPostgresDB and migrate.Up(). SetupSuite wires a real DB; each test uses isolated namespaces. | Tests use context.Background() directly (no testing.T available at DB level) — for new tests prefer t.Context() where t is available. |

## Anti-Patterns

- Adding a method that reads or writes via a.db without wrapping in entutils.TransactingRepo — breaks ctx-carried transaction isolation
- Writing to ChargesSearchV1 — it is a read-only view backed by the underlying charge tables
- Using query.Paginate() after GroupBy — Ent does not support this; paginate manually in Go
- Constructing the adapter without calling config.Validate() — nil client causes a panic at query time
- Adding business logic to adapter methods — adapters must be pure data-access; orchestration belongs in charges/service

## Decisions

- **entutils.TransactingRepo wraps every method body rather than wrapping at the service call site** — Charge advancement mixes reads and writes across multiple adapter calls in a single transaction; each helper must independently rebind to the ctx-carried tx to avoid partial writes.
- **Manual Go-side pagination for ListCustomersToAdvance instead of Ent Paginate** — Ent's generated Paginate() is incompatible with GROUP BY queries; the total count is computed from the full scan and sliced in Go.

## Example: Adding a new query method that must respect transactions

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
