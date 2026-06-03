# service

<!-- archie:ai-start -->

> Business-logic layer for currencies and cost bases. Validates inputs, resolves fiat vs. custom currency enumeration in-memory from GOBL, and delegates persistence to currencies.Adapter via transaction.Run.

## Patterns

**Input validation before transaction** — Every public method calls params.Validate() and returns models.NewGenericValidationError if it fails — before opening a transaction. (`if params.Validate() != nil { return ..., models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate())) }`)
**transaction.Run wrapping adapter calls** — All adapter interactions are wrapped in transaction.Run(ctx, s.adapter, func(ctx) ...) so the service participates in caller-supplied transactions. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.Currency, error) { return s.adapter.CreateCurrency(ctx, params) })`)
**In-memory fiat enumeration via GOBL** — Fiat currencies come from currency.Definitions() (invopop/gobl) filtered to ISO-numeric-only; custom currencies come from the adapter. Combined listing merges both slices with manual in-memory pagination. (`filteredMatchCode, err := lo.FilterErr(currency.Definitions(), func(def *currency.Def, _ int) (bool, error) { if def.ISONumeric == "" { return false, nil }; return matchCode(def.ISOCode.String(), 0) })`)
**Compile-time interface assertion** — var _ currencies.CurrencyService = (*Service)(nil) ensures Service always satisfies the interface. (`var _ currencies.CurrencyService = (*Service)(nil)`)
**Business-logic only — no Ent imports** — The service package must not import openmeter/ent/db; all DB access flows through currencies.Adapter to preserve the layer boundary. (`// imports only: currencies, transaction, models, pagination, gobl, samber/lo — never openmeter/ent/db`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single file implementing currencies.CurrencyService: ListCurrencies (merges fiat+custom in-memory), CreateCurrency, CreateCostBasis (future-only EffectiveFrom defaulting to time.Now()), ListCostBases. | ListCurrencies has two pagination paths — DB-level (custom-only fast path) vs in-memory (fiat or combined); a third filter type must handle both. EffectiveFrom defaulting to time.Now() when nil is service-layer logic, not adapter logic. |
| `service_test.go` | Unit tests using fakeAdapter (in-memory slice impl of currencies.Adapter) and noopDriver. Covers combined/custom-only listing, filter operators (Eq, In, Ne), sort order, and validation errors. | Tests use t.Context() (not context.Background()). New adapter interface methods must be added as panicking stubs in fakeAdapter. |

## Anti-Patterns

- Importing openmeter/ent/db directly — all DB access must go through the currencies.Adapter interface.
- Skipping params.Validate() before calling the adapter.
- Applying pagination logic inside the adapter for combined fiat+custom queries — in-memory merging belongs here.
- Using context.Background() instead of propagating the caller's ctx through transaction.Run.
- Adding temporal business rules (future-date enforcement, defaulting) inside the adapter — belongs in the service.

## Decisions

- **Fiat currency list sourced in-memory from GOBL, not stored in the DB.** — ISO fiat currencies are stable and exhaustive in GOBL; storing them in Postgres would require synchronisation and add write paths with no benefit.
- **EffectiveFrom defaulting and future-date validation lives in the service, not the adapter.** — Adapter is a pure persistence layer; temporal business rules must hold regardless of call path, so they belong in the service.

## Example: A service method that validates input and delegates to the adapter inside a transaction

```
func (s *Service) DeleteCurrency(ctx context.Context, params currencies.DeleteCurrencyInput) error {
	if params.Validate() != nil {
		return models.NewGenericValidationError(fmt.Errorf("invalid input: %w", params.Validate()))
	}
	_, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, s.adapter.DeleteCurrency(ctx, params)
	})
	return err
}
```

<!-- archie:ai-end -->
