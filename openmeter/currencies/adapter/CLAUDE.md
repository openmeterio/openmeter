# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL persistence layer for the currencies domain, implementing the currencies.Adapter interface over the CustomCurrency and CurrencyCostBasis tables. It is the only place that touches entdb for custom currencies and FX cost-basis rates.

## Patterns

**Adapter satisfies currencies.Adapter** — The unexported adapter struct must implement currencies.Adapter; assert it at compile time in both files. (`var _ currencies.Adapter = (*adapter)(nil)`)
**Constructor validates Config** — New(config Config) calls config.Validate() (Client must be non-nil) and returns (currencies.Adapter, error); never expose the concrete struct. (`func New(config Config) (currencies.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Transaction-aware via entutils.TransactingRepo** — Every adapter method wraps its body in entutils.TransactingRepo(ctx, a, func(ctx, tx)...) so it rebinds to a tx carried in ctx. Implement Tx/WithTx/Self for entutils.TxCreator. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (...) { ... })`)
**DB->domain mapping helpers** — Use mapCurrencyFromDB / mapCostBasisFromDB to translate entdb rows into currencies.Currency / currencies.CostBasis; never return entdb types. Normalize times with .In(time.UTC). (`func mapCostBasisFromDB(c *entdb.CurrencyCostBasis) currencies.CostBasis { ... EffectiveFrom: c.EffectiveFrom.In(time.UTC) }`)
**Namespace-scoped queries** — Always filter by Namespace (e.g. customcurrency.Namespace, currencycostbasis.Namespace) from params; multi-tenancy is enforced here, not in the service. (`q := a.db.CustomCurrency.Query().Where(customcurrency.Namespace(params.Namespace))`)
**Constraint errors map to conflict** — On Save, check entdb.IsConstraintError(err) and return models.NewGenericConflictError(...); wrap all other errors with fmt.Errorf("...: %w", err). (`if entdb.IsConstraintError(err) { return currencies.Currency{}, models.NewGenericConflictError(...) }`)
**pkg/filter + pagination helpers** — Apply field filters via filter.ApplyToQuery and prefer q.Paginate(ctx, params.Page) + pagination.MapResult for list results; manual Offset/Limit only when Page is non-zero. (`paged, err := q.Paginate(ctx, params.Page); return pagination.MapResult(paged, mapCostBasisFromDB), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config/New constructor and the entutils.TxCreator plumbing (Tx, WithTx, Self) for transaction propagation. | WithTx must rebuild via entdb.NewTxClientFromRawConfig and carry logger forward; the logger field is currently never set by New. |
| `currencies.go` | Concrete CRUD/list implementations: ListCustomCurrencies, CreateCurrency, CreateCostBasis, ListCostBases, plus the FromDB mappers. | ListCustomCurrencies hand-rolls Offset/Limit (only when Page>0) while ListCostBases uses q.Paginate — keep ordering/default-order logic (entutils.GetOrdering) intact. |

## Anti-Patterns

- Returning entdb.* types instead of mapping to currencies.Currency / currencies.CostBasis
- Bypassing entutils.TransactingRepo or accessing a.db directly outside the tx wrapper
- Querying without a Namespace predicate (breaks tenant isolation)
- Returning a raw constraint error instead of models.NewGenericConflictError
- Exposing the concrete adapter struct instead of the currencies.Adapter interface from New

## Decisions

- **Adapter holds no business rules (no future-date checks, no fiat enumeration)** — All validation and fiat-vs-custom merging lives in the service; the adapter is a thin tenant-scoped persistence boundary.

## Example: Tenant-scoped create with conflict mapping

```
func (a *adapter) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) {
		curr, err := a.db.CustomCurrency.Create().SetNamespace(params.Namespace).SetCode(params.Code).SetName(params.Name).SetSymbol(params.Symbol).Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return currencies.Currency{}, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code))
			}
			return currencies.Currency{}, fmt.Errorf("failed to create currency: %w", err)
		}
		return mapCurrencyFromDB(curr), nil
	})
}
```

<!-- archie:ai-end -->
