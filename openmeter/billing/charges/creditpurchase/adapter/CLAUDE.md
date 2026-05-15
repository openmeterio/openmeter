# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing creditpurchase.Adapter — all DB reads and writes for credit-purchase charges, credit grants, external payments, and invoiced payments. Every method honors the ctx-bound transaction via entutils.TransactingRepo and enforces namespace isolation on every query.

## Patterns

**TransactingRepo on every method (reads and writes)** — Both mutating and read-only methods wrap Ent calls in entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ...) to participate in caller-supplied transactions and guarantee consistent reads across multi-step flows. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) { entity, err := tx.db.ChargeCreditPurchase.Query().Where(...).Only(ctx); ... })`)
**TxCreator + TxUser triad: Tx / WithTx / Self** — adapter implements Tx(ctx) via HijackTx+NewTxDriver, WithTx(ctx, tx) creating a new txClient from TxDriver config, and Self() returning itself — all three are required for entutils.TransactingRepo to rebind to the caller's transaction. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger, metaAdapter: a.metaAdapter} }`)
**Config.Validate() before construction** — New(Config) calls config.Validate() and returns an error if Client, Logger, or MetaAdapter is nil — never construct with zero values. (`func New(config Config) (creditpurchase.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{...}, nil }`)
**Namespace predicate on every query** — Every Ent query includes a namespace WHERE predicate (dbchargecreditpurchase.NamespaceEQ / dbchargecreditpurchase.Namespace) to enforce multi-tenant isolation. (`tx.db.ChargeCreditPurchase.Query().Where(dbchargecreditpurchase.Namespace(input.ChargeID.Namespace), dbchargecreditpurchase.ID(input.ChargeID.ID))`)
**withExpands for conditional edge loading** — Ent edges (WithCreditGrant, WithExternalPayment, WithInvoicedPayment) are loaded only when meta.Expands flag is set via the withExpands helper — never eager-load unconditionally. (`func withExpands(query *db.ChargeCreditPurchaseQuery, expands meta.Expands) *db.ChargeCreditPurchaseQuery { if expands.Has(meta.ExpandRealizations) { query = query.WithCreditGrant().WithExternalPayment().WithInvoicedPayment() }; return query }`)
**Mapper functions in mapper.go** — All Ent-to-domain mapping lives in MapChargeBaseFromDB and MapCreditPurchaseChargeFromDB in mapper.go — adapter methods call these and never inline mapping logic. Edge fields are only read when expands flag is set; missing edges are detected with lo.ErrorsAs[*entdb.NotLoadedError]. (`return MapCreditPurchaseChargeFromDB(dbCreditPurchase, meta.ExpandNone)`)
**Bidirectional cursor pagination with sort reversal** — ListFundedCreditActivities reverses the sort order when input.Before != nil, then calls slices.Reverse(items) after fetching — both steps are required for correct bidirectional pagination. The method is also exported as a package-level function for callers that have a raw *db.Client. (`if input.Before != nil { query = query.Order(ByGrantedAt(sql.OrderAsc()), ...); ... slices.Reverse(items) } else { query = query.Order(ByGrantedAt(sql.OrderDesc()), ...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter constructor, Config struct with Validate(), and Tx/WithTx/Self triad for entutils integration. | Never omit WithTx or Self — TransactingRepo requires both for tx rebinding. MetaAdapter must not be nil. |
| `charge.go` | CRUD for ChargeCreditPurchase rows: UpdateCharge, CreateCharge, GetByID, GetByIDs, ListCharges. | CreateCharge calls metaAdapter.RegisterCharges inside the same TransactingRepo transaction — this ordering must be preserved. |
| `creditgrant.go` | Creates ChargeCreditPurchaseCreditGrant rows linking a charge to a ledger transaction group. | GrantedAt must be stored in UTC: input.GrantedAt.In(time.UTC). |
| `funded_credit_activity.go` | Cursor-based pagination over credit grants joined to credit purchases; both an adapter method and a package-level function ListFundedCreditActivities. | Before-cursor path reverses sort order AND calls slices.Reverse(items) — omitting either step breaks pagination ordering. |
| `mapper.go` | MapChargeBaseFromDB and MapCreditPurchaseChargeFromDB translate Ent rows to domain types; edge fields are only read when the expands flag is set. | Always check lo.ErrorsAs[*entdb.NotLoadedError] when reading edges — missing NotLoadedError check panics instead of returning a clear error. |
| `payment.go` | Create/Update for external and invoiced payment settlement rows. | All four methods wrap in TransactingRepo — never bypass even for simple single-row saves. |

## Anti-Patterns

- Using a.db directly inside a method body instead of tx.db from TransactingRepo — falls off the caller's transaction.
- Calling Ent edges without setting up withExpands — results in NotLoadedError panics in mapper.
- Omitting namespace predicate from queries — returns cross-tenant data.
- Inlining Ent-to-domain mapping instead of calling Map* functions in mapper.go.
- Constructing adapter without calling Config.Validate() — causes nil-pointer panics at first use.

## Decisions

- **TransactingRepo wraps every method (reads and writes) rather than only writes.** — Charge advancement mixes reads and writes across multiple helpers; consistent reads inside a caller tx prevent phantom reads between steps.
- **ListFundedCreditActivities is also exported as a package-level function, not only an adapter method.** — Tests and potential callers need to call it with a raw *db.Client without constructing a full adapter.
- **Edge loading (WithCreditGrant, etc.) is gated behind meta.Expands rather than always eager-loaded.** — Avoiding unnecessary joins keeps list/get queries fast; callers explicitly opt in to realizations data.

## Example: Adding a new mutating adapter method

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) SomeUpdate(ctx context.Context, in creditpurchase.SomeInput) (creditpurchase.SomeResult, error) {
	if err := in.Validate(); err != nil {
		return creditpurchase.SomeResult{}, err
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.SomeResult, error) {
		entity, err := tx.db.ChargeCreditPurchase.UpdateOneID(in.ID).
			Where(dbchargecreditpurchase.NamespaceEQ(in.Namespace)).
			Save(ctx)
// ...
```

<!-- archie:ai-end -->
