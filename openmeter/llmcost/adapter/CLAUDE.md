# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for the llmcost domain, implementing llmcost.Adapter against the LLMCostPrice table. Owns all SQL access for global prices (namespace IS NULL) and per-namespace manual overrides, plus their soft-delete and effective-period resolution semantics.

## Patterns

**Config-validated constructor returning interface** — New(Config) returns (llmcost.Adapter, error); Config.Validate() collects missing Client/Logger into errors.Join before constructing. Compile-time assertions var _ models.Validator = (*Config)(nil) and var _ llmcost.Adapter = (*adapter)(nil) enforce the contracts. (`func New(config Config) (llmcost.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger}, nil }`)
**Transaction plumbing via entutils** — adapter implements Tx (HijackTx + entutils.NewTxDriver), WithTx (rebind via entdb.NewTxClientFromRawConfig), and Self. Every write/list wraps its body in entutils.TransactingRepo / TransactingRepoWithNoValue so it rebinds to the tx in ctx. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (llmcost.Price, error) { ... })`)
**Validate input before any query** — Each method calls input.Validate() as the first statement inside the transaction closure (or at top of ResolvePrice) before touching a.db. (`if err := input.Validate(); err != nil { return llmcost.Price{}, err }`)
**Soft delete everywhere** — All reads filter pricedb.DeletedAtIsNil(); deletes/replacements SetDeletedAt(clock.Now()) instead of removing rows. CreateOverride soft-deletes the prior active manual override before inserting the new one. (`Where(pricedb.DeletedAtIsNil()).Where(pricedb.NamespaceIsNil())`)
**Namespace-aware price resolution ordering** — ResolvePrice ORs NamespaceEQ(input.Namespace) with NamespaceIsNil(), filters EffectiveFrom<=at and (EffectiveTo IS NULL or >at), then Orders ByNamespace(desc) so a namespace override wins over the global row. (`Order(pricedb.ByNamespace(sql.OrderDesc()), pricedb.ByEffectiveFrom(sql.OrderDesc())).First(ctx)`)
**Domain error mapping** — entdb.IsNotFound -> llmcost.NewPriceNotFoundError / NewPriceOverrideNotFoundError; entdb.IsConstraintError -> models.NewGenericConflictError. Never leak raw ent errors as not-found. (`if entdb.IsNotFound(err) { return llmcost.Price{}, llmcost.NewPriceOverrideNotFoundError(input.ID) }`)
**filter.ApplyToQuery for list filtering** — List methods apply optional FilterString inputs via filter.ApplyToQuery(query, input.X, pricedb.FieldX) and order via entutils.GetOrdering(input.Order) with an explicit OrderBy switch defaulting to model ID. (`query = filter.ApplyToQuery(query, input.Provider, pricedb.FieldProvider)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config, New, the adapter struct, and Tx/WithTx/Self transaction plumbing. | Requires both Client and Logger; do not fall back to slog.Default(). WithTx must rebuild from *tx.GetConfig(). |
| `price.go` | All Adapter methods: ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides, UpsertGlobalPrice. | ResolvePrice intentionally does NOT use a transaction (read-only, uses clock.Now() when input.At is nil). ListOverrides hard-codes source=manual and must not also apply a user source filter (creates contradictory WHERE). |
| `mapping.go` | mapPriceFromEntity (db.LLMCostPrice -> llmcost.Price) and decimalOrZero helper. | Optional pricing fields (CacheRead/CacheWrite/Reasoning) are stored non-null; a zero decimal means 'not set' and is dropped to a nil pointer via !entity.X.IsZero(). decimalOrZero reverses this on writes. |

## Anti-Patterns

- Using a raw *entdb.Client directly instead of wrapping in entutils.TransactingRepo, breaking tx-awareness in ctx.
- Hard-deleting rows or skipping pricedb.DeletedAtIsNil() filters, exposing soft-deleted prices.
- Returning entdb.IsNotFound errors directly instead of mapping to llmcost.NewPrice*NotFoundError.
- Adding a user-supplied source filter inside ListOverrides (already constrained to source=manual).
- Persisting optional pricing pointers without decimalOrZero, or mapping zero decimals back to non-nil pointers.

## Decisions

- **Global prices use NULL namespace; overrides set the namespace and source=manual.** — A single LLMCostPrice table holds both tiers; resolution prefers the namespace override via ByNamespace(desc) ordering without a separate table.
- **ResolvePrice bypasses the transaction wrapper.** — It is a hot read-only path on the metering/cost lookup; avoiding HijackTx reduces overhead.
- **UpsertGlobalPrice updates the current (EffectiveTo IS NULL) row in place rather than versioning.** — The sync reconciler refreshes canonical system prices repeatedly; in-place update keeps one current global row per provider/model.

## Example: Tx-aware adapter write with input validation, soft-delete of prior override, and error mapping

```
func (a *adapter) CreateOverride(ctx context.Context, input llmcost.CreateOverrideInput) (llmcost.Price, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (llmcost.Price, error) {
		if err := input.Validate(); err != nil {
			return llmcost.Price{}, err
		}
		_, _ = a.db.LLMCostPrice.Update().
			Where(pricedb.DeletedAtIsNil(), pricedb.NamespaceEQ(input.Namespace), pricedb.ProviderEQ(string(input.Provider)), pricedb.ModelIDEQ(input.ModelID), pricedb.SourceEQ(string(llmcost.PriceSourceManual))).
			SetDeletedAt(clock.Now()).Save(ctx)
		entity, err := a.db.LLMCostPrice.Create().SetNamespace(input.Namespace).SetSource(string(llmcost.PriceSourceManual)).Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				err = models.NewGenericConflictError(err)
			}
			return llmcost.Price{}, fmt.Errorf("failed to create override: %w", err)
		}
// ...
```

<!-- archie:ai-end -->
