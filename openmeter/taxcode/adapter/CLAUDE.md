# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer (taxcode.Repository) for tax codes and per-namespace organization default tax codes. All DB access is transaction-aware and namespace-scoped; this is the only place that talks to entdb.Client for taxcode tables.

## Patterns

**Repository constructor with validated Config** — New(Config) validates Client + Logger via models.Validator and returns taxcode.Repository, never the concrete *adapter. (`func New(config Config) (taxcode.Repository, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger}, nil }`)
**Tx/WithTx/Self transaction trio** — adapter implements Tx (HijackTx), WithTx (rebind via NewTxClientFromRawConfig), and Self so entutils.TransactingRepo can rebind to the ctx-carried tx. (`func (a *adapter) WithTx(ctx, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Wrap every method body in TransactingRepo** — Each repo method calls input.Validate() then entutils.TransactingRepo(ctx, a, func...) (or TransactingRepoWithNoValue for void) so it joins an existing tx. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) { ... })`)
**Namespace + soft-delete filtering** — Reads filter taxcodedb.Namespace(...) and (unless IncludeDeleted) taxcodedb.DeletedAtIsNil(); deletes are soft via SetDeletedAt(clock.Now()). (`query.Where(taxcodedb.Namespace(input.Namespace)).Where(taxcodedb.DeletedAtIsNil())`)
**Map Ent entity -> domain via Map*FromEntity** — Entities are converted with MapTaxCodeFromEntity / mapOrganizationDefaultTaxCodesFromEntity; never return *db.TaxCode upward. Expand edges loaded conditionally and read via *OrErr(). (`return MapTaxCodeFromEntity(entity)`)
**Translate Ent errors to domain errors** — db.IsNotFound -> taxcode.New*NotFoundError; db.IsConstraintError -> models.NewGenericConflictError. Raw Ent errors are never leaked. (`if db.IsNotFound(err) { return ..., taxcode.NewTaxCodeNotFoundError(input.ID) }`)
**Upsert via OnConflict on namespace partial-unique** — Org defaults upsert uses OnConflict(ConflictColumns(FieldNamespace), ConflictWhere(IsNull(FieldDeletedAt))).UpdateNewValues() then re-reads via Get to honor Expand. (`Create().SetNamespace(...).OnConflict(sql.ConflictColumns(orgdefaultsdb.FieldNamespace), sql.ConflictWhere(sql.IsNull(orgdefaultsdb.FieldDeletedAt))).UpdateNewValues().Exec(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config, New, and the Tx/WithTx/Self transaction plumbing | Return taxcode.Repository not *adapter; WithTx must rebuild db from the raw tx config, not reuse a.db. |
| `taxcode.go` | CRUD for TaxCode plus GetTaxCodeByAppMapping | GetTaxCodeByAppMapping uses a raw JSONB '@>' containment query and sorts system-managed first then by CreatedAt/ID — preserve that ordering; UpdateTaxCode clears AppMappings when input list is empty. |
| `organizationdefaulttaxcodes.go` | Get/Upsert org default tax codes with edge expansion | Expansion edges loaded only when input.Expand.* is set; mapping uses InvoicingTaxCodeOrErr()/CreditGrantTaxCodeOrErr() and errors if the requested edge wasn't loaded. |
| `mapping.go` | MapTaxCodeFromEntity: single source of TaxCode entity->domain conversion | AppMappings read via lo.FromPtr (nullable); Metadata wrapped with models.NewMetadata; nil entity returns an error. |

## Anti-Patterns

- Returning *db.TaxCode or any Ent type to callers instead of mapping to taxcode domain types
- Querying a.db directly without entutils.TransactingRepo, breaking transactional composition with the service layer
- Omitting the Namespace or DeletedAtIsNil filter, leaking cross-namespace or soft-deleted rows
- Leaking raw Ent errors instead of converting via db.IsNotFound / db.IsConstraintError to taxcode/models errors
- Hard-deleting tax codes instead of soft-delete via SetDeletedAt(clock.Now())

## Decisions

- **App-mapping lookup is a JSONB containment query (app_mappings @> [...]) plus in-Go stable sort** — A single tax code can carry multiple app mappings and multiple codes can match; system-managed seeds must win deterministically over user duplicates.
- **Org default tax codes use a partial-unique upsert keyed on namespace where deleted_at is null** — Exactly one active default row per namespace, kept idempotent and stable (CreatedAt/ID unchanged) across repeated upserts.

## Example: Adapter CRUD method: validate, transact, query namespace-scoped, translate errors, map

```
func (a *adapter) GetTaxCode(ctx context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) {
		entity, err := a.db.TaxCode.Query().
			Where(taxcodedb.Namespace(input.Namespace)).
			Where(taxcodedb.ID(input.ID)).
			Where(taxcodedb.DeletedAtIsNil()).
			Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return taxcode.TaxCode{}, taxcode.NewTaxCodeNotFoundError(input.ID)
			}
			return taxcode.TaxCode{}, fmt.Errorf("failed to get tax code: %w", err)
// ...
```

<!-- archie:ai-end -->
