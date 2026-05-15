# currencies

<!-- archie:ai-start -->

> Manages custom currencies and fiat-exchange cost bases for billing namespaces. Two-layer domain: adapter/ owns Ent/PostgreSQL persistence with full TransactingRepo discipline; service/ merges in-memory GOBL fiat enumeration with DB-stored custom currencies.

## Patterns

**TransactingRepo wrapping every DB method** — Every adapter method body is wrapped with entutils.TransactingRepo or TransactingRepoWithNoValue to honor the ctx-bound transaction. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (Currency, error) { row, err := tx.db.Currency.Create()...; return mapCurrencyFromDB(row), err })`)
**Tx / WithTx / Self triad on adapter** — adapter implements entutils.TxCreator (Tx method) and TxUser[*adapter] (WithTx + Self) so callers can join or start transactions uniformly. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Input Validate() before transaction.Run** — Service calls params.Validate() before calling transaction.Run; constraint errors from the DB are a last resort, not the primary validation gate. (`if err := params.Validate(); err != nil { return Currency{}, err }; return transaction.Run(ctx, s.adapter, func(ctx context.Context) (Currency, error) { return s.adapter.CreateCurrency(ctx, params) })`)
**entdb.IsConstraintError → GenericConflictError** — Adapter maps Ent constraint violations to models.NewGenericConflictError, not raw Ent errors. (`if entdb.IsConstraintError(err) { return Currency{}, models.NewGenericConflictError(err) }`)
**mapXFromDB mapper functions** — Separate top-level mapper functions (mapCurrencyFromDB, mapCostBasisFromDB) convert Ent rows to domain types — not methods on Ent types. (`func mapCurrencyFromDB(row *entdb.Currency) Currency { return Currency{Code: row.Code, ...} }`)
**In-memory fiat enumeration via GOBL** — Service lists fiat currencies from GOBL at runtime (not from DB); in-memory merge and pagination of fiat + custom sets belongs in the service, not the adapter. (`fiatCurrencies := gobl.AllCurrencies(); result := mergeAndPaginate(fiatCurrencies, customCurrencies, params.Page)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/currencies/adapter.go` | Defines Adapter interface (CurrenciesAdapter + entutils.TxCreator) and CurrenciesAdapter sub-interface. | Adapter embeds TxCreator — any new implementation must provide Tx/WithTx/Self. |
| `openmeter/currencies/models.go` | All domain types (Currency, CostBasis) and all input types with Validate() implementations. | All input structs implement models.Validator via var _ models.Validator = (*XInput)(nil) — add the assertion for any new input type. |
| `openmeter/currencies/service.go` | Defines CurrencyService interface — the public facade for HTTP handlers. | Service interface must stay in sync with the adapter methods it delegates to; fiat-vs-custom logic lives here not in the adapter. |
| `openmeter/currencies/adapter/adapter.go` | Concrete Ent adapter; Config struct, Validate(), New() constructor, Tx/WithTx/Self. | Config.Validate() checks adapter dependencies are non-nil — replicate for any new dependency added to Config. |
| `openmeter/currencies/adapter/currencies.go` | Per-method DB implementations for ListCustomCurrencies, CreateCurrency, CreateCostBasis, ListCostBases. | Every method must wrap with entutils.TransactingRepo — omitting it bypasses the caller's transaction. |

## Anti-Patterns

- Calling a.db.X directly inside a method body without entutils.TransactingRepo — bypasses ctx-bound transactions.
- Importing openmeter/ent/db in the service layer — all DB access must go through the currencies.Adapter interface.
- Storing fiat currencies in the DB — they are enumerated in-memory from GOBL and merged at query time in the service.
- Returning raw Ent errors to callers — always map constraint errors to models.GenericConflictError and not-found to models.GenericNotFoundError.
- Constructing *adapter directly instead of calling adapter.New(Config{...}) — Validate() in New() enforces required deps.

## Decisions

- **Fiat currency list sourced in-memory from GOBL, not stored in DB.** — Fiat currencies are a stable enumeration; storing them would require migrations on standard updates and create drift risk.
- **Tx/WithTx/Self triad on the adapter struct, not a shared TxWrapper type.** — Currencies is a small domain; a single adapter file is simpler and mirrors the pattern used across other small domain adapters.
- **EffectiveFrom defaulting and future-date validation in service, not adapter.** — Business rules around time-validity belong in the service layer; the adapter only persists what it is given.

## Example: Adding a new adapter method for a new currencies entity

```
import (
	"context"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) CreateFoo(ctx context.Context, params currencies.CreateFooInput) (currencies.Foo, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Foo, error) {
		row, err := tx.db.Foo.Create().SetNamespace(params.Namespace).SetCode(params.Code).Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return currencies.Foo{}, models.NewGenericConflictError(err)
			}
			return currencies.Foo{}, err
// ...
```

<!-- archie:ai-end -->
