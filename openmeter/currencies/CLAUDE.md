# currencies

<!-- archie:ai-start -->

> Manages custom currencies and fiat-exchange cost bases for billing namespaces. Two-layer domain: adapter/ owns Ent/PostgreSQL persistence with full TransactingRepo discipline; service/ merges in-memory GOBL fiat enumeration with DB-stored custom currencies.

## Patterns

**Input Validate() before transaction.Run** — Service calls params.Validate() before transaction.Run; DB constraint errors are a last resort, not the primary validation gate. (`if err := params.Validate(); err != nil { return Currency{}, err }; return transaction.Run(ctx, s.adapter, func(ctx) (Currency, error) { return s.adapter.CreateCurrency(ctx, params) })`)
**TransactingRepo wrapping every adapter method** — Every adapter method body is wrapped with entutils.TransactingRepo/TransactingRepoWithNoValue to honor the ctx-bound transaction. (`return entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter) (Currency, error) { row, err := tx.db.Currency.Create()...; return mapCurrencyFromDB(row), err })`)
**entdb.IsConstraintError → GenericConflictError** — Adapter maps Ent constraint violations to models.NewGenericConflictError, never raw Ent errors. (`if entdb.IsConstraintError(err) { return Currency{}, models.NewGenericConflictError(err) }`)
**In-memory fiat enumeration via GOBL** — Fiat currencies are listed from GOBL at runtime (not the DB); in-memory merge and pagination of fiat + custom sets lives in the service. (`fiatCurrencies := gobl.AllCurrencies(); result := mergeAndPaginate(fiatCurrencies, customCurrencies, params.Page)`)
**models.Validator assertion on every input type** — All input structs declare var _ models.Validator = (*XInput)(nil) and implement Validate() with errors.Join. (`var _ models.Validator = (*ListCurrenciesInput)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Adapter (CurrenciesAdapter + entutils.TxCreator) and the CurrenciesAdapter sub-interface. | Adapter embeds TxCreator — any implementation must provide Tx/WithTx/Self. |
| `models.go` | All domain types (Currency, CostBasis) and input types (ListCurrenciesInput, CreateCurrencyInput, CreateCostBasisInput, ListCostBasesInput) with Validate() and CurrencyType enum. | Add var _ models.Validator assertion for any new input type; Rate uses alpacadecimal and must be positive. |
| `service.go` | Defines CurrencyService interface — the public facade for HTTP handlers. | Keep in sync with adapter methods it delegates to; fiat-vs-custom logic lives in service/, not the adapter. |

## Anti-Patterns

- Calling a.db.X directly without entutils.TransactingRepo — bypasses ctx-bound transactions
- Importing openmeter/ent/db in the service layer — all DB access must go through the currencies.Adapter interface
- Storing fiat currencies in the DB — they are enumerated in-memory from GOBL and merged at query time
- Returning raw Ent errors — map constraint errors to GenericConflictError and not-found to GenericNotFoundError
- Adding temporal business rules (future-date enforcement, defaulting) inside the adapter — they belong in the service

## Decisions

- **Fiat currency list sourced in-memory from GOBL, not stored in the DB** — Fiat currencies are a stable enumeration; storing them would require migrations on standard updates and create drift risk.
- **Tx/WithTx/Self triad inlined on the adapter struct, not a shared TxWrapper** — Currencies is a small domain; a single adapter file mirrors the pattern used across other small domain adapters.

<!-- archie:ai-end -->
