# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for the root charges facade. Implements charges.Adapter (transaction plumbing) and charges.ChargesSearchAdapter (read-only queries over the chargessearchv1 view) — it never owns per-charge-type state; that lives in flatfee/usagebased/creditpurchase adapters.

## Patterns

**Validate-then-TransactingRepo wrapper** — Every adapter method calls input.Validate() then wraps its body in entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter)...) so it rebinds to the tx carried in ctx. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (...) { dbCharges, err := tx.db.ChargesSearchV1.Query()... })`)
**Tx/WithTx/Self transaction trio** — adapter implements Tx (HijackTx), WithTx (rebuild from raw config via entdb.NewTxClientFromRawConfig), and Self so transaction.Run and TransactingRepo can drive it. (`func (a *adapter) Tx(ctx) (...) { txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly:false}); return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil }`)
**Reads target the chargessearchv1 view** — All search queries go through tx.db.ChargesSearchV1 with dbchargessearchv1 predicates (Namespace, IDIn, StatusIn, DeletedAtIsNil), never the concrete charge tables. (`tx.db.ChargesSearchV1.Query().Where(dbchargessearchv1.Namespace(input.Namespace)).Where(dbchargessearchv1.IDIn(input.IDs...))`)
**Stable result ordering via entutils.InIDOrder** — GetByIDs reorders DB rows to match the requested ID slice using entutils.InIDOrder with an InIDOrderAccessor wrapper, enforcing namespace ownership. (`resultsInOrder, err := entutils.InIDOrder(input.Namespace, input.IDs, withIDAccessor(dbCharges))`)
**Manual pagination after GroupBy** — ListCustomersToAdvance uses GroupBy(namespace, customer_id) which cannot Paginate, so it slices results by page.Offset()/page.Limit() manually after a Namespace+CustomerID ordered scan. (`query.Order(dbchargessearchv1.ByNamespace(), dbchargessearchv1.ByCustomerID()).GroupBy(FieldNamespace, FieldCustomerID).Scan(ctx, &results)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config{Client,Logger}+Validate, New() returning charges.Adapter, and the Tx/WithTx/Self transaction trio. | Both Client and Logger are required in Validate(); WithTx must rebuild the ent client from raw tx config, not reuse a.db. |
| `search.go` | Implements ChargesSearchAdapter: GetByIDs, ListCharges, ListCustomersToAdvance, plus mapChargeSearchToChargeWithType and the searchResultIDAccessor. | ListCustomersToAdvance filters StatusNotIn(ChargeStatusFinal, ChargeStatusDeleted) and AdvanceAfterLTE; nil AdvanceAfter rows are excluded. OrderBy switch only supports id/service_period.from/billing_period.from/created_at. |
| `search_test.go` | Suite-based integration test (InitPostgresDB + migrate.OMMigrationsConfig) inserting ChargeFlatFee rows directly to exercise the search view. | Inserts via dbClient.ChargeFlatFee.Create (concrete table) but reads via the chargessearchv1 view, so the view must be migrated. |

## Anti-Patterns

- Querying concrete charge tables (ChargeFlatFee, etc.) for reads instead of the ChargesSearchV1 view.
- Accessing a.db directly inside a method instead of the tx-bound client from TransactingRepo.
- Returning rows in DB order from GetByIDs instead of InIDOrder request order.
- Adding per-charge-type lifecycle logic here — this adapter only does search/tx plumbing.

## Decisions

- **Search is served by a dedicated chargessearchv1 view rather than UNIONing concrete tables.** — Lets the root facade list/filter heterogeneous charge types (id, customer, status, advance_after) in one indexed query.
- **ListCustomersToAdvance returns deduped customer IDs via GroupBy, not charges.** — The advance worker iterates customers, so the adapter collapses many due charges per customer into one row.

<!-- archie:ai-end -->
