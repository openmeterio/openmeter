# service

<!-- archie:ai-start -->

> Business-logic layer for currencies and cost bases. Validates inputs, resolves fiat vs. custom currency enumeration in-memory from GOBL, and delegates persistence to currencies.Adapter via transaction.Run.

## Patterns

**Input validation before transaction** — Every public method calls params.Validate() and returns models.NewGenericValidationError if it fails — before opening a transaction. (`if params.Validate() != nil { return pagination.Result[currencies.Currency]{}, models.NewGenericValidationError(fmt.Errorf("invalid input parameters: %w", params.Validate())) }`)
**transaction.Run wrapping adapter calls** — All adapter interactions are wrapped in transaction.Run(ctx, s.adapter, func(ctx) ...) so the service participates in caller-supplied transactions. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (currencies.Currency, error) { return s.adapter.CreateCurrency(ctx, params) })`)
**In-memory fiat enumeration via GOBL** — Fiat currencies come from currency.Definitions() (invopop/gobl) filtered to ISO-numeric-only; custom currencies come from the adapter. Combined listing merges both slices and applies manual pagination in-memory. (`filteredMatchCode, err := lo.FilterErr(currency.Definitions(), func(def *currency.Def, _ int) (bool, error) { if def.ISONumeric == "" { return false, nil }; return matchCode(def.ISOCode.String(), 0) })`)
**Compile-time interface assertion** — var _ currencies.CurrencyService = (*Service)(nil) at package top ensures Service always satisfies the interface. (`var _ currencies.CurrencyService = (*Service)(nil)`)
**Business-logic only — no Ent imports** — The service package must not import openmeter/ent/db; all DB access must flow through currencies.Adapter to preserve the layer boundary. (`// service.go imports only: currencies, transaction, models, pagination, gobl, samber/lo — never openmeter/ent/db`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single file implementing currencies.CurrencyService: ListCurrencies (merges fiat+custom in-memory), CreateCurrency, CreateCostBasis (enforces future-only EffectiveFrom with default to time.Now()), ListCostBases. | ListCurrencies has two distinct pagination paths — DB-level (custom-only fast path) vs. in-memory (fiat or combined); adding a third filter type must handle both correctly. EffectiveFrom defaulting to time.Now() when nil is service-layer business logic, not adapter logic. |
| `service_test.go` | Unit tests using a fakeAdapter (implements currencies.Adapter with in-memory slice) and noopDriver (transaction.Driver no-op). Tests cover combined/custom-only listing paths, filter operators (Eq, In, Ne), sort order, and validation errors. | Tests use t.Context() (not context.Background()). fakeAdapter.ListCustomCurrencies applies Code filter to simulate DB-level filtering — new adapter methods added to the interface must be added as panicking stubs here. |

## Anti-Patterns

- Importing openmeter/ent/db directly — all DB access must go through the currencies.Adapter interface
- Skipping params.Validate() before calling the adapter — constraint errors from the DB are harder to diagnose than early validation errors
- Applying pagination logic inside the adapter for combined fiat+custom queries — the adapter only handles DB rows; in-memory merging and pagination belong here
- Using context.Background() instead of propagating the caller's ctx through transaction.Run
- Adding temporal business rules (future-date enforcement, defaulting) inside the adapter — belongs in the service

## Decisions

- **Fiat currency list sourced in-memory from GOBL, not stored in the DB** — ISO fiat currencies are stable and exhaustive in the GOBL library; storing them in Postgres would require synchronisation and add write paths with no benefit.
- **EffectiveFrom defaulting and future-date validation lives in the service, not the adapter** — Adapter is a pure persistence layer; temporal business rules must be enforced regardless of which code path calls the adapter, so they belong in the service.

## Example: Adding a new service method that validates input and delegates to the adapter inside a transaction

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) DeleteCurrency(ctx context.Context, params currencies.DeleteCurrencyInput) error {
	if params.Validate() != nil {
		return models.NewGenericValidationError(fmt.Errorf("invalid input: %w", params.Validate()))
	}
	_, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, s.adapter.DeleteCurrency(ctx, params)
// ...
```

<!-- archie:ai-end -->
