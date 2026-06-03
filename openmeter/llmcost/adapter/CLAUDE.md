# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing llmcost.Adapter: persists global (namespace IS NULL, source='system') and per-namespace manual override (source='manual') rows in the llmcostprice table with soft-delete throughout. All mutations honor the ctx-bound Ent transaction via entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping on every mutating method** — Each DB-writing method body is wrapped with entutils.TransactingRepo or TransactingRepoWithNoValue to rebind to any caller transaction; read-only ResolvePrice intentionally skips this. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (llmcost.Price, error) { ... a.db.LLMCostPrice.Create()... })`)
**Soft-delete via SetDeletedAt(clock.Now())** — All deletions set DeletedAt; all queries filter pricedb.DeletedAtIsNil(). Never hard-delete. (`a.db.LLMCostPrice.UpdateOneID(id).SetDeletedAt(clock.Now()).Exec(ctx)`)
**Namespace discrimination by predicate** — Global rows use pricedb.NamespaceIsNil(); overrides use pricedb.NamespaceEQ(ns) + pricedb.SourceEQ('manual'). Never mix predicates or add a user source filter to ListOverrides. (`query.Where(pricedb.NamespaceEQ(ns), pricedb.SourceEQ("manual"))`)
**decimalOrZero for optional pricing fields** — Optional decimal pointers (CacheReadPerToken, CacheWritePerToken, ReasoningPerToken) pass through decimalOrZero before SetX so nil becomes zero, avoiding NULL in non-nullable columns. (`SetCacheReadPerToken(decimalOrZero(input.Pricing.CacheReadPerToken))`)
**ResolvePrice single-query override precedence** — ResolvePrice fetches namespace AND global rows in one query ordered by namespace DESC so namespace-set rows sort first, returning the first hit — no fallback query. (`Order(pricedb.ByNamespace(sql.OrderDesc()), pricedb.ByEffectiveFrom(sql.OrderDesc())).First(ctx)`)
**UpsertGlobalPrice query-first in-place update** — Checks for an existing open-ended global row (EffectiveToIsNil); UpdateOneID in place if found else Create. Globals are not soft-delete-then-insert. (`existing, _ := a.db.LLMCostPrice.Query().Where(pricedb.NamespaceIsNil(), pricedb.EffectiveToIsNil()).First(ctx)`)
**Config.Validate() before construction** — New(Config) calls config.Validate() (Config implements models.Validator) and returns an error before building the adapter. (`func New(config Config) (llmcost.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor, Config validation, Tx/WithTx/Self transaction plumbing required by TransactingRepo. | WithTx must call entdb.NewTxClientFromRawConfig and return a new *adapter backed by txClient.Client() — never reuse the original db field across transactions. |
| `mapping.go` | Maps *db.LLMCostPrice to llmcost.Price; optional pricing fields use an IsZero() guard before setting a pointer. | A zero alpacadecimal from the DB means unset, not explicit zero — check IsZero() before assigning a pointer. |
| `price.go` | All llmcost.Adapter methods: ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides, UpsertGlobalPrice. | ListOverrides hardcodes SourceEQ('manual') — do not add a caller source filter. DeleteOverride validates SourceEQ('manual') so system/global prices cannot be deleted this way. |

## Anti-Patterns

- Hard-deleting rows — all deletions go through SetDeletedAt(clock.Now()).
- Calling a.db directly inside a helper without entutils.TransactingRepo wrapping.
- Mixing pricedb.NamespaceIsNil() and pricedb.NamespaceEQ() in a single ListPrices query path.
- Adding a source filter argument to ListOverrides (already constrained to source='manual').
- Editing files under openmeter/ent/db/ instead of regenerating via make generate.

## Decisions

- **Soft-delete for all price records.** — Price history is required for billing audit trails; hard deletes would destroy the audit trail for past invoice line resolutions.
- **ResolvePrice reads directly without TransactingRepo.** — It is read-only with no side effects; the ORDER BY namespace DESC trick makes it a single-query hot path in the billing request path.

## Example: A new write method honoring the ctx transaction

```
func (a *adapter) ArchivePrice(ctx context.Context, id string) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, a *adapter) error {
		return a.db.LLMCostPrice.UpdateOneID(id).
			SetDeletedAt(clock.Now()).
			Exec(ctx)
	})
}
```

<!-- archie:ai-end -->
