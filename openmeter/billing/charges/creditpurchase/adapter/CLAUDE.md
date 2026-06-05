# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for credit-purchase charges, their credit grants, and external/invoiced payment settlements. Implements the creditpurchase.Adapter / CreditGrantAdapter interfaces over the ChargeCreditPurchase* Ent entities.

## Patterns

**Constructor returns interface, validates Config** — New(Config) returns creditpurchase.Adapter; Config.Validate() requires Client, Logger, MetaAdapter. Compile-time check via var _ creditpurchase.Adapter = (*adapter)(nil). (`func New(config Config) (creditpurchase.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Every method wraps body in entutils.TransactingRepo** — Read and write methods bind to the ctx-carried tx via entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter)...). tx.db is the tx-aware client; never use a.db directly inside a method body. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) { query := tx.db.ChargeCreditPurchase.Query()... })`)
**Validate input before any DB access** — Each method calls input.Validate() / chargeID.Validate() and returns the zero value + error before touching the DB. (`if err := input.Validate(); err != nil { return creditpurchase.Charge{}, err }`)
**DB->domain mapping via Map*FromDB in mapper.go** — All Ent entities convert to domain through MapChargeBaseFromDB / MapCreditPurchaseChargeFromDB; expand-gated edges (CreditGrant, ExternalPayment, InvoicedPayment) error with lo.ErrorsAs[*entdb.NotLoadedError] if not eager-loaded. (`if expands.Has(meta.ExpandRealizations) { dbCreditGrant, err := dbEntity.Edges.CreditGrantOrErr(); ... }`)
**Register charges with meta adapter on create** — CreateCharge persists the row then calls tx.metaAdapter.RegisterCharges with ChargeTypeCreditPurchase so the meta layer tracks the new charge by ID + UniqueReferenceID. (`err = tx.metaAdapter.RegisterCharges(ctx, meta.RegisterChargesInput{ Type: meta.ChargeTypeCreditPurchase, Charges: []meta.IDWithUniqueReferenceID{...} })`)
**Keyset pagination via explicit cursor predicates** — ListFundedCreditActivities orders by (GrantedAt, CreditPurchase.CreatedAt, ChargeID) and builds tie-broken Or/And predicates (fundedCreditActivityAfterPredicate/BeforePredicate); fetches Limit+1 to compute hasMore and reverses items for Before. (`query.Order(dbchargecreditpurchasecreditgrant.ByGrantedAt(sql.OrderDesc()), ...ByCreditPurchaseField(...CreatedAt, sql.OrderDesc()), ...ByChargeID(sql.OrderDesc()))`)
**Tx plumbing trio: Tx/WithTx/Self** — adapter implements the entutils transacting contract: Tx hijacks a tx, WithTx rebuilds the adapter from raw tx config, Self returns the receiver. (`func (a *adapter) WithTx(ctx, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), ...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config, New constructor, the adapter struct, and Tx/WithTx/Self transaction plumbing | WithTx must carry logger and metaAdapter forward; Tx uses ReadOnly:false |
| `charge.go` | Charge CRUD: CreateCharge, UpdateCharge, GetByID, GetByIDs, ListCharges, and withExpands edge loader | CreateCharge must call metaAdapter.RegisterCharges; GetByIDs uses entutils.InIDOrder to preserve input order; ExpandRealizations loads three edges together |
| `creditgrant.go` | CreateCreditGrant - persists a ChargeCreditPurchaseCreditGrant and returns a ledgertransaction.TimedGroupReference | GrantedAt is forced to time.UTC on both write and read |
| `funded_credit_activity.go` | ListFundedCreditActivities keyset-paginated query plus after/before cursor predicates | package-level ListFundedCreditActivities takes a raw *db.Client; cursor tie-breaking spans three columns; Before path reverses results |
| `mapper.go` | Map*FromDB converters from Ent entities to creditpurchase domain types | realization edges are only mapped when expands.Has(meta.ExpandRealizations); not-loaded edges surface as errors, not nil |
| `payment.go` | Create/Update for External and Invoiced payment settlements via payment.CreateExternal/UpdateExternal/CreateInvoiced/UpdateInvoiced helpers | builder mutation is delegated to the payment models package; only ChargeID is set locally before handing off |
| `funded_credit_activity_test.go` | Postgres-backed suite for cursor pagination, currency filter, and as-of filter | uses testutils.InitPostgresDB + migrate.OMMigrationsConfig; inserts raw Ent rows rather than going through the service |

## Anti-Patterns

- Using a.db directly inside a method instead of the tx-bound tx.db from entutils.TransactingRepo
- Mapping realization edges without first checking expands.Has(meta.ExpandRealizations) / using *OrErr
- Skipping input.Validate()/chargeID.Validate() before DB access
- Creating a charge without calling metaAdapter.RegisterCharges
- Dropping logger or metaAdapter when rebuilding the adapter in WithTx

## Decisions

- **Every adapter method re-binds to the ambient transaction via entutils.TransactingRepo even helpers handed a raw *entdb.Client** — Keeps Ent access transaction-aware regardless of caller, per the charges adapter convention in AGENTS.md
- **Realization data (credit grant, external/invoiced payment) is loaded only under ExpandRealizations** — Avoids three extra joins on list/get hot paths and forces callers to opt in to the heavier read

## Example: Adapter create method: validate, transact, persist via Ent builder, register in meta layer, map to domain

```
func (a *adapter) CreateCharge(ctx context.Context, in creditpurchase.CreateChargeInput) (creditpurchase.Charge, error) {
	if err := in.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditpurchase.Charge, error) {
		create := tx.db.ChargeCreditPurchase.Create().SetNamespace(in.Namespace).SetCreditAmount(in.Intent.CreditAmount).SetSettlement(in.Intent.Settlement).SetStatusDetailed(creditpurchase.StatusCreated)
		create, err := chargemeta.Create(create, chargemeta.CreateInput{Namespace: in.Namespace, Intent: in.Intent.Intent, Status: meta.ChargeStatusCreated})
		if err != nil { return creditpurchase.Charge{}, err }
		dbCreditPurchase, err := create.Save(ctx)
		if err != nil { return creditpurchase.Charge{}, err }
		if err := tx.metaAdapter.RegisterCharges(ctx, meta.RegisterChargesInput{Namespace: in.Namespace, Type: meta.ChargeTypeCreditPurchase, Charges: []meta.IDWithUniqueReferenceID{{ID: dbCreditPurchase.ID, UniqueReferenceID: dbCreditPurchase.UniqueReferenceID}}}); err != nil { return creditpurchase.Charge{}, err }
		return MapCreditPurchaseChargeFromDB(dbCreditPurchase, meta.ExpandNone)
	})
}
```

<!-- archie:ai-end -->
