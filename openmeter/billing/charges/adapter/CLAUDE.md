# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing charges.Adapter and charges.ChargesSearchAdapter; provides all DB access for the charges domain (search, advancement candidates) with strict transaction rebinding via entutils.TransactingRepo on every exported method.

## Patterns

**TransactingRepo on every method** — Every exported adapter method body wraps in entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter) ...) so it rebinds to the ctx-carried Ent transaction. Never call tx.db.Foo() outside this wrapper. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (T, error) { return tx.db.ChargesSearchV1.Query()... })`)
**Tx / WithTx / Self triad** — adapter implements TxCreator via Tx() (HijackTx + NewTxDriver), TxUser[T] via WithTx() (NewTxClientFromRawConfig), and Self() — all three required for TransactingRepo to rebind or self-start transactions. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**Config.Validate() before construction** — New(Config) calls config.Validate() and errors if Client or Logger is nil before constructing the adapter struct. Never construct adapter directly. (`func New(config Config) (charges.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger}, nil }`)
**Interface compliance assertions** — Each implemented interface has a compile-time var _ assertion at file scope. (`var _ charges.Adapter = (*adapter)(nil)
var _ charges.ChargesSearchAdapter = (*adapter)(nil)`)
**Manual Go-side pagination for GroupBy queries** — Ent's Paginate() is incompatible with GroupBy; ListCustomersToAdvance fetches all grouped results via Scan() then slices manually in Go. (`err := query.GroupBy(fields...).Scan(ctx, &results); start := page.Offset(); pageResults := results[start:end]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, and the adapter struct with Tx/WithTx/Self for transaction lifecycle. Shared by all query files. | Never pass a.db directly to a helper — always go through tx *adapter inside TransactingRepo so the Ent client is bound to the current transaction. |
| `search.go` | Implements GetByIDs, ListCharges, ListCustomersToAdvance using the ChargesSearchV1 Ent view; shows namespace-filter and GroupBy+manual-pagination workaround. | ChargesSearchV1 is a read-only Ent view — never write through it. GroupBy results cannot use query.Paginate(); paginate manually in Go. |
| `search_test.go` | Integration tests against a real Postgres via testutils.InitPostgresDB and migrate.Up(), each using isolated namespaces. | Tests use context.Background() at DB setup; prefer t.Context() in new test methods where *testing.T is available. |

## Anti-Patterns

- Adding a method that reads/writes via a.db without wrapping in entutils.TransactingRepo — breaks ctx-carried transaction isolation
- Writing to ChargesSearchV1 — it is a read-only Ent view backed by underlying charge tables
- Using query.Paginate() after GroupBy — Ent does not support this; paginate manually in Go
- Constructing the adapter without config.Validate() — nil client panics at query time
- Adding business logic (orchestration, state machine calls) to adapter methods — adapters must be pure data-access

## Decisions

- **entutils.TransactingRepo wraps every method body rather than at the service call site** — Charge advancement mixes reads and writes across multiple adapter calls in one transaction; each helper must independently rebind to the ctx-carried tx to avoid partial writes.
- **Manual Go-side pagination for ListCustomersToAdvance instead of Ent Paginate** — Ent's generated Paginate() is incompatible with GROUP BY; total count is computed from the full scan and sliced in Go.

## Example: Adding a new query method that respects ctx-carried transactions

```
func (a *adapter) GetActiveByCustomer(ctx context.Context, input charges.GetActiveInput) ([]charges.ChargeSearchItem, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]charges.ChargeSearchItem, error) {
        rows, err := tx.db.ChargesSearchV1.Query().
            Where(dbchargessearchv1.Namespace(input.Namespace)).
            Where(dbchargessearchv1.CustomerID(input.CustomerID)).
            Where(dbchargessearchv1.DeletedAtIsNil()).
            All(ctx)
        if err != nil { return nil, err }
        return lo.Map(rows, func(r *db.ChargesSearchV1, _ int) charges.ChargeSearchItem { return mapChargeSearchToChargeWithType(r) }), nil
    })
}
```

<!-- archie:ai-end -->
