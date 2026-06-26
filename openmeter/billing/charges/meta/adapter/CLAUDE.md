# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter implementing the meta.Adapter interface for charge-meta rows: registering charges (flatfee/usagebased/creditpurchase) and soft-deleting them. Primary constraint: every write must run inside a transaction rebound from ctx, and the meta.Adapter contract (Tx/WithTx/Self) must stay intact for entutils transaction wiring.

## Patterns

**Transaction-aware repo wrapping** — Every mutating method wraps its body in entutils.TransactingRepoWithNoValue(ctx, a, func(ctx, tx *adapter)...) so the adapter rebinds to the tx carried in ctx instead of using the raw injected client. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { ... tx.db.Charge.Create()... })`)
**Input.Validate() before any DB access** — Each method calls in.Validate() and returns early before touching Ent. Validation lives on the meta input types, not the adapter. (`func (a *adapter) RegisterCharges(ctx, in meta.RegisterChargesInput) error { if err := in.Validate(); err != nil { return err } ... }`)
**Standard Config/Validate/New constructor** — Construction goes through Config{Client, Logger} with a Validate() that requires both fields non-nil, then New(config) returns the meta.Adapter interface (not the concrete type). (`func New(config Config) (meta.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{...}, nil }`)
**Interface assertion at package scope** — var _ meta.Adapter = (*adapter)(nil) guarantees the concrete adapter satisfies the meta.Adapter interface at compile time. (`var _ meta.Adapter = (*adapter)(nil)`)
**Tx/WithTx/Self transaction plumbing trio** — The adapter implements Tx (HijackTx + entutils.NewTxDriver), WithTx (rebinds via entdb.NewTxClientFromRawConfig), and Self — the exact trio entutils.TransactingRepo* expects. (`func (a *adapter) WithTx(ctx, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**clock.Now() for all timestamps** — Created/deleted timestamps use pkg/clock.Now() (freezable in tests), never time.Now(). (`SetCreatedAt(clock.Now()) / SetDeletedAt(clock.Now())`)
**Charge-type switch sets the FK column** — RegisterCharges switches on in.Type to set the matching foreign-key column (ChargeFlatFeeID / ChargeUsageBasedID / ChargeCreditPurchaseID); an unknown type returns an error rather than silently skipping. (`switch in.Type { case meta.ChargeTypeFlatFee: create = create.SetChargeFlatFeeID(charge.ID); ... default: return nil, fmt.Errorf("unknown charge type: %s", in.Type) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, Validate, New, the adapter struct (db, logger), and the Tx/WithTx/Self transaction interface methods that satisfy meta.Adapter. | Tx uses HijackTx with ReadOnly:false; WithTx must rebuild the client from tx.GetConfig() via entdb.NewTxClientFromRawConfig — do not reuse the original a.db inside a tx scope. |
| `charges.go` | Implements the two write methods RegisterCharges (bulk create via slicesx.MapWithErr + Charge.CreateBulk) and DeleteRegisteredCharge (soft delete via UpdateOneID + SetDeletedAt). | DeleteRegisteredCharge filters with chargedb.DeletedAtIsNil() and chargedb.Namespace(in.Namespace) — always scope by namespace and exclude already-deleted rows; deletes are soft (SetDeletedAt), never hard. |

## Anti-Patterns

- Calling tx.db / a.db directly without an entutils.TransactingRepo* wrapper, breaking ctx-bound transaction reuse
- Using time.Now() instead of clock.Now() for created/deleted timestamps
- Hard-deleting charge rows instead of setting DeletedAt (soft delete)
- Omitting the namespace and DeletedAtIsNil filters on updates/deletes, allowing cross-tenant or double-delete writes
- Returning the concrete *adapter from New or skipping the var _ meta.Adapter assertion, decoupling the impl from the meta contract

## Decisions

- **Methods accept a single meta input struct and validate it before DB access.** — Keeps validation co-located with the domain contract in meta/ and ensures no malformed write reaches Ent.
- **Adapter implements its own Tx/WithTx/Self trio rather than embedding a shared base.** — entutils.TransactingRepo* requires these exact methods to rebind the adapter onto the ctx transaction; explicit implementation keeps the charge-meta adapter transaction-aware even when handed a raw *entdb.Client.

## Example: A transaction-aware mutating adapter method validating input and bulk-creating Ent rows

```
func (a *adapter) RegisterCharges(ctx context.Context, in meta.RegisterChargesInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		creates, err := slicesx.MapWithErr(in.Charges, func(charge meta.IDWithUniqueReferenceID) (*db.ChargeCreate, error) {
			create := tx.db.Charge.Create().
				SetNamespace(in.Namespace).
				SetType(in.Type).
				SetID(charge.ID).
				SetNillableUniqueReferenceID(charge.UniqueReferenceID).
				SetCreatedAt(clock.Now())
			switch in.Type {
			case meta.ChargeTypeFlatFee:
				create = create.SetChargeFlatFeeID(charge.ID)
// ...
```

<!-- archie:ai-end -->
