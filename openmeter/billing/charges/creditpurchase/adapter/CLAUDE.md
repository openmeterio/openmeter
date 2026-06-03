# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing creditpurchase.Adapter — all DB reads/writes for credit-purchase charges, credit grants, external payments, and invoiced payments. Every method honors the ctx-bound transaction via entutils.TransactingRepo and enforces namespace isolation.

## Patterns

**TransactingRepo on every method (reads and writes)** — Both mutating and read-only methods wrap Ent calls in entutils.TransactingRepo to participate in caller transactions and guarantee consistent reads across multi-step flows. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) { entity, err := tx.db.ChargeCreditPurchase.Query().Where(...).Only(ctx); ... })`)
**Tx / WithTx / Self triad** — Implement Tx(ctx) via HijackTx+NewTxDriver, WithTx(ctx, tx) creating a txClient from TxDriver config, and Self(); all three are required for TransactingRepo to rebind. (`func (a *adapter) WithTx(ctx, tx) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger, metaAdapter: a.metaAdapter} }`)
**Config.Validate() before construction** — New(Config) validates Client, Logger, MetaAdapter non-nil and returns an error — never construct with zero values. (`func New(config Config) (creditpurchase.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{...}, nil }`)
**Namespace predicate on every query** — Every Ent query includes a namespace WHERE predicate to enforce multi-tenant isolation. (`tx.db.ChargeCreditPurchase.Query().Where(dbchargecreditpurchase.Namespace(input.ChargeID.Namespace), dbchargecreditpurchase.ID(input.ChargeID.ID))`)
**withExpands for conditional edge loading** — Edges (WithCreditGrant, WithExternalPayment, WithInvoicedPayment) load only when meta.Expands is set via withExpands — never eager-load unconditionally. (`func withExpands(q *db.ChargeCreditPurchaseQuery, e meta.Expands) *db.ChargeCreditPurchaseQuery { if e.Has(meta.ExpandRealizations) { q = q.WithCreditGrant().WithExternalPayment().WithInvoicedPayment() }; return q }`)
**Mapper functions in mapper.go** — All Ent-to-domain mapping lives in MapChargeBaseFromDB/MapCreditPurchaseChargeFromDB; missing edges detected via lo.ErrorsAs[*entdb.NotLoadedError]. (`return MapCreditPurchaseChargeFromDB(dbCreditPurchase, meta.ExpandNone)`)
**Bidirectional cursor pagination with sort reversal** — ListFundedCreditActivities reverses sort order when input.Before != nil, then slices.Reverse(items) after fetching — both steps required. (`if input.Before != nil { query = query.Order(ByGrantedAt(sql.OrderAsc()), ...); ...; slices.Reverse(items) } else { query = query.Order(ByGrantedAt(sql.OrderDesc()), ...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor, Config + Validate(), Tx/WithTx/Self triad. | Never omit WithTx or Self — TransactingRepo requires both; MetaAdapter must not be nil. |
| `charge.go` | CRUD for ChargeCreditPurchase rows: UpdateCharge, CreateCharge, GetByID, GetByIDs, ListCharges. | CreateCharge calls metaAdapter.RegisterCharges inside the same TransactingRepo transaction — preserve this ordering. |
| `creditgrant.go` | Creates ChargeCreditPurchaseCreditGrant rows linking a charge to a ledger transaction group. | GrantedAt must be stored in UTC: input.GrantedAt.In(time.UTC). |
| `funded_credit_activity.go` | Cursor pagination over credit grants joined to credit purchases; adapter method + package-level ListFundedCreditActivities. | Before-cursor path reverses sort order AND calls slices.Reverse(items) — omitting either breaks ordering. |
| `mapper.go` | MapChargeBaseFromDB and MapCreditPurchaseChargeFromDB translate Ent rows to domain types. | Check lo.ErrorsAs[*entdb.NotLoadedError] when reading edges — missing check panics instead of returning a clear error. |
| `payment.go` | Create/Update for external and invoiced payment settlement rows. | All methods wrap in TransactingRepo — never bypass even for single-row saves. |

## Anti-Patterns

- Using a.db directly inside a method body instead of tx.db from TransactingRepo — falls off the caller's transaction.
- Calling Ent edges without withExpands — NotLoadedError panics in the mapper.
- Omitting the namespace predicate from queries — returns cross-tenant data.
- Inlining Ent-to-domain mapping instead of calling Map* functions in mapper.go.
- Constructing the adapter without Config.Validate() — nil-pointer panics at first use.

## Decisions

- **TransactingRepo wraps every method (reads and writes).** — Charge advancement mixes reads and writes across helpers; consistent reads in a caller tx prevent phantom reads between steps.
- **ListFundedCreditActivities is also exported as a package-level function.** — Tests and callers need to invoke it with a raw *db.Client without constructing a full adapter.
- **Edge loading is gated behind meta.Expands rather than always eager-loaded.** — Avoiding unnecessary joins keeps list/get queries fast; callers opt in to realizations data.

## Example: Adding a new mutating adapter method

```
func (a *adapter) SomeUpdate(ctx context.Context, in creditpurchase.SomeInput) (creditpurchase.SomeResult, error) {
	if err := in.Validate(); err != nil { return creditpurchase.SomeResult{}, err }
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.SomeResult, error) {
		entity, err := tx.db.ChargeCreditPurchase.UpdateOneID(in.ID).Where(dbchargecreditpurchase.NamespaceEQ(in.Namespace)).Save(ctx)
		if err != nil { return creditpurchase.SomeResult{}, err }
		return MapCreditPurchaseChargeFromDB(entity, meta.ExpandNone)
	})
}
```

<!-- archie:ai-end -->
