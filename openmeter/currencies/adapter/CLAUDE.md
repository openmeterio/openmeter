# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing currencies.Adapter for custom currencies and cost bases. All DB access is transaction-aware via entutils.TransactingRepo; the adapter implements the TxCreator+TxUser triad to start or join caller-supplied transactions.

## Patterns

**TransactingRepo wrapping every method** — Every public adapter method body is wrapped with entutils.TransactingRepo(ctx, a, func(ctx, tx) ...) so the ctx-bound Ent transaction is honoured. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) { curr, err := tx.db.CustomCurrency.Create()... })`)
**Tx / WithTx / Self triad** — Tx() hijacks a new pg transaction via a.db.HijackTx, WithTx() rebinds via entdb.NewTxClientFromRawConfig, Self() returns itself. All three required by TransactingRepo. (`func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) { txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false}); return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil }`)
**Config struct + Validate + New constructor** — Adapter is constructed via New(Config) with Config.Validate() guard; callers never instantiate the adapter struct directly. (`func New(config Config) (currencies.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client}, nil }`)
**Compile-time interface assertion** — var _ currencies.Adapter = (*adapter)(nil) declared in both adapter.go and currencies.go to catch interface drift. (`var _ currencies.Adapter = (*adapter)(nil)`)
**entdb.IsConstraintError to GenericConflictError** — On Ent constraint violations wrap with models.NewGenericConflictError so the HTTP layer maps to 409 automatically. (`if entdb.IsConstraintError(err) { return currencies.Currency{}, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code)) }`)
**mapXFromDB mapper functions** — DB row to domain conversion is always in package-level mapXFromDB functions (mapCurrencyFromDB, mapCostBasisFromDB); never inline struct construction in query handlers. Mappers normalize time to UTC. (`func mapCostBasisFromDB(c *entdb.CurrencyCostBasis) currencies.CostBasis { return currencies.CostBasis{..., EffectiveFrom: c.EffectiveFrom.In(time.UTC)} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, the adapter struct, and Tx/WithTx/Self transaction plumbing — the skeleton every adapter method depends on. | Do not add business logic here; it is strictly wiring. If Self() or WithTx() is removed or its signature changes, TransactingRepo breaks silently. |
| `currencies.go` | Implements all currencies.Adapter methods (ListCustomCurrencies, CreateCurrency, CreateCostBasis, ListCostBases) plus the DB to domain mappers. | ListCustomCurrencies uses manual Offset/Limit pagination; ListCostBases uses q.Paginate — keep both consistent when extending. Mappers must normalize time to UTC (c.EffectiveFrom.In(time.UTC)). |

## Anti-Patterns

- Calling a.db.X directly inside a method body without entutils.TransactingRepo — bypasses the ctx-bound Ent transaction and produces partial writes.
- Adding service-layer validation or business logic inside the adapter — belongs in openmeter/currencies/service.
- Constructing *adapter directly instead of calling New(Config).
- Returning raw Ent errors to callers — always wrap or map to models.Generic* errors.
- Storing *entdb.Tx as a struct field instead of using the TxDriver/TransactingRepo pattern.

## Decisions

- **Tx/WithTx/Self triad inlined on the adapter struct rather than a separate TxWrapper type.** — entutils.TransactingRepo requires the repository itself to implement TxCreator; inlining avoids an extra indirection layer and matches all other domain adapters.
- **Separate mapper functions rather than methods on Ent types.** — Keeps generated Ent types isolated from domain types; Ent schema changes only require updating the mapper, not every call site.

## Example: Adding a new adapter method that writes to the DB

```
func (a *adapter) UpdateCurrency(ctx context.Context, params currencies.UpdateCurrencyInput) (currencies.Currency, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) {
		curr, err := tx.db.CustomCurrency.UpdateOneID(params.ID).SetName(params.Name).Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return currencies.Currency{}, models.NewGenericConflictError(err)
			}
			return currencies.Currency{}, fmt.Errorf("update currency: %w", err)
		}
		return mapCurrencyFromDB(curr), nil
	})
}
```

<!-- archie:ai-end -->
