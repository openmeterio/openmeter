# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing llmcost.Adapter: persists global (namespace IS NULL, source='system') and per-namespace manual override (source='manual') rows in the llmcostprice table with soft-delete semantics throughout. All mutations must honor the ctx-bound Ent transaction via entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping on every mutating method** — Every method body that writes to the DB is wrapped with entutils.TransactingRepo or entutils.TransactingRepoWithNoValue so the adapter rebinds to any caller-supplied ctx transaction. Read-only ResolvePrice skips this but must be documented. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (llmcost.Price, error) { ... a.db.LLMCostPrice.Create()... })`)
**Soft-delete via SetDeletedAt(clock.Now())** — All deletions set DeletedAt, never hard-delete. All queries filter with pricedb.DeletedAtIsNil(). (`a.db.LLMCostPrice.UpdateOneID(id).SetDeletedAt(clock.Now()).Exec(ctx)`)
**Namespace discrimination by predicate** — Global prices use pricedb.NamespaceIsNil(). Per-namespace overrides use pricedb.NamespaceEQ(ns) + pricedb.SourceEQ('manual'). Never mix predicates or add a user-supplied source filter to ListOverrides (already constrained). (`query.Where(pricedb.NamespaceIsNil()) // global; query.Where(pricedb.NamespaceEQ(ns), pricedb.SourceEQ("manual")) // override`)
**decimalOrZero for optional pricing fields** — Optional decimal pointer fields (CacheReadPerToken, CacheWritePerToken, ReasoningPerToken) use decimalOrZero(*alpacadecimal.Decimal) before SetX so nil pointers become zero-value, avoiding NULL in non-nullable columns. (`SetCacheReadPerToken(decimalOrZero(input.Pricing.CacheReadPerToken))`)
**ResolvePrice single-query override precedence via ORDER BY** — ResolvePrice fetches namespace AND global rows in one query, orders by namespace DESC (namespace-set rows sort first), returns the first hit. No second query for fallback. (`Order(pricedb.ByNamespace(sql.OrderDesc()), pricedb.ByEffectiveFrom(sql.OrderDesc())).First(ctx)`)
**UpsertGlobalPrice query-first in-place update** — UpsertGlobalPrice checks for an existing open-ended global row (EffectiveToIsNil). If found, UpdateOneID in place; otherwise Create. Does not soft-delete-then-insert for globals. (`existing, err := a.db.LLMCostPrice.Query().Where(pricedb.NamespaceIsNil(), pricedb.EffectiveToIsNil()).First(ctx); if existing != nil { UpdateOneID(existing.ID)... } else { Create()... }`)
**Config struct with Validate() before construction** — Constructor accepts Config; Config implements models.Validator. New() calls config.Validate() and returns an error before building the adapter. (`var _ models.Validator = (*Config)(nil); func New(config Config) (llmcost.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor, Config validation, Tx/WithTx/Self transaction plumbing required by TransactingRepo. | WithTx must call entdb.NewTxClientFromRawConfig and return a new *adapter backed by txClient.Client() — never reuse the original db field across transactions. |
| `mapping.go` | Maps *db.LLMCostPrice entity to llmcost.Price domain type. Optional pricing fields use IsZero() guard before setting pointer. | A zero alpacadecimal value from the DB means the field was not set, not that it is explicitly zero — check IsZero() before assigning a pointer. |
| `price.go` | All llmcost.Adapter method implementations: ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides, UpsertGlobalPrice. | ListOverrides hardcodes SourceEQ('manual') — do not add a caller-supplied source filter on top. DeleteOverride validates SourceEQ('manual') before allowing deletion so global (system) prices cannot be deleted this way. |

## Anti-Patterns

- Hard-deleting rows — all deletions must go through SetDeletedAt(clock.Now())
- Calling a.db directly inside a helper without entutils.TransactingRepo wrapping — falls off the ctx-bound Ent transaction
- Mixing pricedb.NamespaceIsNil() and pricedb.NamespaceEQ() in a single ListPrices query path
- Adding a source filter argument to ListOverrides (already constrained to source='manual')
- Editing files under openmeter/ent/db/ — regenerate via make generate after schema changes

## Decisions

- **Soft-delete for all price records** — Price history is required for billing audit trails. Hard deletes would destroy the audit trail for past invoice line resolutions.
- **ResolvePrice reads directly without TransactingRepo** — It is intentionally read-only with no side-effects; the ORDER BY namespace DESC trick makes it a single-query hot path used in the billing request path.

## Example: Add a new write method that must honor the ctx transaction

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
