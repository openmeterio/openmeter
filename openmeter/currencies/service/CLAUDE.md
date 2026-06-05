# service

<!-- archie:ai-start -->

> Business-logic layer for the currencies domain, implementing currencies.CurrencyService. It validates input, merges DB-backed custom currencies with ISO fiat currencies from gobl, and orchestrates transactions over the adapter.

## Patterns

**Service satisfies currencies.CurrencyService** — Service struct wraps a single currencies.Adapter; assert interface conformance and construct via New(adapter). (`var _ currencies.CurrencyService = (*Service)(nil); func New(adapter currencies.Adapter) *Service`)
**Validate-then-Run guard** — Every public method first calls params.Validate() and returns models.NewGenericValidationError on failure, then wraps the work in transaction.Run(ctx, s.adapter, ...). (`if params.Validate() != nil { return ..., models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate())) }`)
**Fiat/custom merge with in-memory pagination** — ListCurrencies derives includeCustom/includeFiat from params.FilterType; custom-only delegates DB pagination to the adapter, otherwise it enumerates (fetching custom with Page{} cleared), sorts, and paginates slices in-memory. (`if includeCustom && !includeFiat { return s.adapter.ListCustomCurrencies(ctx, params) }`)
**Fiat sourced from gobl currency.Definitions** — Fiat currencies come from currency.Definitions(); rows with empty ISONumeric (crypto/non-ISO) are filtered out, and the code filter is applied via params.Code.LoFilterPredicate(). (`if def.ISONumeric == "" { return false, nil }; return matchCode(def.ISOCode.String(), 0)`)
**Service-owned business rules** — Domain invariants live here, e.g. CreateCostBasis rejects a non-future EffectiveFrom and defaults a nil EffectiveFrom to time.Now() before calling the adapter. (`if params.EffectiveFrom != nil && !params.EffectiveFrom.After(now) { return ..., models.NewGenericValidationError(...) }`)
**Sort honors OrderBy/Order** — In-memory results are sorted via slices.SortFunc comparing Name or Code per params.OrderBy, negating for sortx.OrderDesc, matching the adapter's DB ordering. (`slices.SortFunc(items, func(a, b currencies.Currency) int { ... if params.Order == sortx.OrderDesc { return -result } })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Implements ListCurrencies (fiat/custom merge), CreateCurrency, CreateCostBasis (future-date rule + default), ListCostBases. | ListCurrencies clears allParams.Page before fetching custom for the combined path; forgetting that re-applies DB pagination and corrupts the in-memory merge. CreateCostBasis computes now=time.Now() inside transaction.Run. |
| `service_test.go` | Pure unit tests using a fakeAdapter + noopDriver (no DB) to exercise combined and custom-only paths. | fakeAdapter only implements ListCustomCurrencies; the other methods panic("not implemented"). Tests rely on real gobl fiat data, so USD/EUR/GBP must remain valid ISO codes; use t.Context(). |

## Anti-Patterns

- Calling the adapter without first validating params and wrapping in transaction.Run
- Putting business rules (future-date checks, fiat filtering) in the adapter instead of the service
- Paginating the custom fetch in the combined path instead of clearing Page for the in-memory merge
- Returning raw validation errors instead of models.NewGenericValidationError
- Including non-ISO/crypto currencies (empty ISONumeric) in fiat enumeration

## Decisions

- **Custom-only listing delegates to DB-level pagination; mixed/fiat listing paginates in-memory** — Fiat currencies are a static gobl-sourced set not stored in Postgres, so merged results must be enumerated and sliced after sorting rather than via SQL OFFSET/LIMIT.
- **EffectiveFrom must be in the future and defaults to now** — Cost-basis FX rates represent forward-effective conversion rates; back-dating would silently rewrite historical billing math.

## Example: Validate, transaction.Run, branch on currency type

```
func (s *Service) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	if params.Validate() != nil {
		return pagination.Result[currencies.Currency]{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate()))
	}
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[currencies.Currency], error) {
		includeCustom := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeCustom
		includeFiat := params.FilterType == nil || *params.FilterType == currencies.CurrencyTypeFiat
		if includeCustom && !includeFiat {
			return s.adapter.ListCustomCurrencies(ctx, params)
		}
		// ...enumerate custom (Page cleared) + fiat, sort, paginate in-memory...
	})
}
```

<!-- archie:ai-end -->
