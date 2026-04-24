# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing creditpurchase.Adapter — all DB reads and writes for credit-purchase charges, credit grants, external payments, and invoiced payments. Every method must honor the ctx-bound transaction via entutils.TransactingRepo.

## Patterns

**TransactingRepo on every write** — Every mutating method wraps its Ent calls in entutils.TransactingRepo(ctx, a, func(...) ...) so it rebinds to any transaction already carried in ctx rather than using the raw *entdb.Client. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) { ... tx.db.ChargeCreditPurchase... })`)
**TransactingRepo on reads too** — Even read-only queries (GetByID, GetByIDs, ListCharges) use entutils.TransactingRepo to participate in caller transactions for consistent reads. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) { entity, err := tx.db.ChargeCreditPurchase.Query()... })`)
**WithTx + Self for TransactingRepo rebind** — adapter implements WithTx(ctx, tx) and Self() required by entutils.TxUser so TransactingRepo can rebind to a new txClient derived from the TxDriver. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), ...} }`)
**Config.Validate() before construction** — New() calls config.Validate() and returns an error if any required field (Client, Logger, MetaAdapter) is nil — never construct with zero values. (`func New(config Config) (creditpurchase.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**withExpands for edge loading** — Ent edge loading (WithCreditGrant, WithExternalPayment, WithInvoicedPayment) is gated behind a withExpands helper that checks meta.Expands flags — never eager-load unconditionally. (`func withExpands(query *db.ChargeCreditPurchaseQuery, expands meta.Expands) *db.ChargeCreditPurchaseQuery { if expands.Has(meta.ExpandRealizations) { query = query.WithCreditGrant()... } return query }`)
**Mapper functions in mapper.go** — All Ent-to-domain mapping lives in Map* functions in mapper.go (MapChargeBaseFromDB, MapCreditPurchaseChargeFromDB). Adapter methods call these; they never inline mapping logic. (`return MapCreditPurchaseChargeFromDB(dbCreditPurchase, meta.ExpandNone)`)
**Namespace predicate on every query** — Every query includes a namespace WHERE predicate (dbchargecreditpurchase.NamespaceEQ / dbchargecreditpurchase.Namespace) to enforce multi-tenant isolation. (`Where(dbchargecreditpurchase.Namespace(input.ChargeID.Namespace), dbchargecreditpurchase.ID(input.ChargeID.ID))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter constructor, Config validation, Tx/WithTx/Self for entutils integration. | Never omit WithTx or Self — TransactingRepo requires both for tx rebinding. |
| `charge.go` | CRUD for ChargeCreditPurchase rows: UpdateCharge, CreateCharge, GetByID, GetByIDs, ListCharges. | CreateCharge also calls metaAdapter.RegisterCharges inside the same transaction — ordering matters. |
| `creditgrant.go` | Creates ChargeCreditPurchaseCreditGrant rows linking a charge to a ledger transaction group. | GrantedAt must be stored in UTC (entity.GrantedAt.In(time.UTC)). |
| `funded_credit_activity.go` | Cursor-based pagination over credit grants joined to credit purchases; exposes ListFundedCreditActivities as both adapter method and package-level function. | Before-cursor reverses sort order then slices.Reverse(items) — both steps are required for correct bidirectional pagination. |
| `mapper.go` | MapChargeBaseFromDB and MapCreditPurchaseChargeFromDB translate Ent rows to domain types; edge fields are only read when expands flag is set. | Missing NotLoadedError check when mapping edges will panic; use lo.ErrorsAs[*entdb.NotLoadedError]. |
| `payment.go` | Create/Update for external and invoiced payment settlement rows. | All four methods wrap in TransactingRepo — never bypass even for simple saves. |

## Anti-Patterns

- Using a.db directly inside a method body instead of the tx.db from TransactingRepo — falls off the caller's transaction.
- Calling Ent edges without setting up withExpands — results in NotLoadedError panics in mapper.
- Omitting namespace predicate from queries — returns cross-tenant data.
- Inlining Ent-to-domain mapping instead of using Map* functions in mapper.go.
- Constructing adapter with a nil Client or MetaAdapter without calling Validate() — causes nil-pointer panics at first use.

## Decisions

- **TransactingRepo wraps every method (reads and writes) rather than only writes.** — Charge advancement mixes reads and writes across multiple helpers; consistent reads inside a caller tx prevent phantom reads between steps.
- **ListFundedCreditActivities is also exported as a package-level function (not only an adapter method).** — Tests and potential callers need to call it with a raw *db.Client without constructing a full adapter — the function signature enables both uses.
- **Edge loading (WithCreditGrant, etc.) is gated behind meta.Expands rather than always eager-loaded.** — Avoiding unnecessary joins keeps list/get queries fast; callers explicitly opt in to realizations data.

## Example: Adding a new mutating adapter method

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) SomeUpdate(ctx context.Context, in creditpurchase.SomeInput) (creditpurchase.SomeResult, error) {
	if err := in.Validate(); err != nil {
		return creditpurchase.SomeResult{}, err
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.SomeResult, error) {
		entity, err := tx.db.ChargeCreditPurchase.UpdateOneID(in.ID).
			Where(dbchargecreditpurchase.NamespaceEQ(in.Namespace)).
			// ... set fields ...
			Save(ctx)
// ...
```

<!-- archie:ai-end -->
