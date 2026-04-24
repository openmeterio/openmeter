# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing llmcost.Adapter: persists global prices (namespace IS NULL) and per-namespace manual overrides (source='manual') in the llmcostprice table, with soft-delete semantics throughout.

## Patterns

**TransactingRepo wrapping** — Every mutating method body is wrapped with entutils.TransactingRepo or entutils.TransactingRepoWithNoValue so the adapter rebinds to any ctx-bound transaction. Methods that are intentionally read-only (ResolvePrice) skip wrapping but must be documented. (`entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (T, error) { ... })`)
**Soft-delete via DeletedAt** — All queries filter with pricedb.DeletedAtIsNil(). Deletions set DeletedAt via SetDeletedAt(clock.Now()), never hard-delete rows. (`a.db.LLMCostPrice.UpdateOneID(id).SetDeletedAt(clock.Now()).Exec(ctx)`)
**Config struct with Validate()** — Constructor accepts a Config struct; Config implements models.Validator. New() calls config.Validate() and returns an error before constructing the adapter. (`var _ models.Validator = (*Config)(nil); func New(config Config) (llmcost.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } }`)
**Namespace discrimination** — Global prices use NamespaceIsNil(). Overrides use NamespaceEQ(namespace). Never mix: ListPrices adds NamespaceIsNil(), ListOverrides adds NamespaceEQ+SourceEQ('manual'). (`query.Where(pricedb.NamespaceIsNil()) // global; query.Where(pricedb.NamespaceEQ(ns)) // override`)
**decimalOrZero for optional pricing fields** — Optional decimal fields (CacheReadPerToken, CacheWritePerToken, ReasoningPerToken) use decimalOrZero(*Decimal) when persisting so nil pointers become zero, avoiding NULL in non-nullable columns. (`SetCacheReadPerToken(decimalOrZero(input.Pricing.CacheReadPerToken))`)
**ResolvePrice uses ORDER BY namespace DESC to prefer overrides** — ResolvePrice fetches both global and namespace rows in one query and orders by namespace DESC so namespace-set rows sort before NULL, returning the override without a second query. (`pricedb.ByNamespace(sql.OrderDesc()), pricedb.ByEffectiveFrom(sql.OrderDesc())`)
**UpsertGlobalPrice in-place update** — UpsertGlobalPrice does a query-first: if an existing open-ended global row exists it updates in place; otherwise it inserts. It does not soft-delete-then-insert for globals. (`existing, err := a.db.LLMCostPrice.Query().Where(...EffectiveToIsNil()).First(ctx); if existing != nil { UpdateOneID(existing.ID)... } else { Create()... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor, Config validation, Tx/WithTx/Self transaction plumbing for TransactingRepo | WithTx must use entdb.NewTxClientFromRawConfig and return a new *adapter backed by txClient.Client() — do not reuse the original db field |
| `mapping.go` | Maps *db.LLMCostPrice entity to llmcost.Price domain type; converts zero alpacadecimal to nil pointer for optional fields | Optional pricing fields (CacheRead, CacheWrite, Reasoning) use IsZero() guard before setting pointer — a zero value means the field was not provided, not that it is explicitly zero |
| `price.go` | All llmcost.Adapter method implementations against Ent; contains ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides, UpsertGlobalPrice | ListOverrides hardcodes SourceEQ('manual') — do not add a user-supplied source filter on top or the WHERE clauses contradict; DeleteOverride checks source='manual' before allowing deletion (global prices cannot be deleted this way) |

## Anti-Patterns

- Hard-deleting rows — all deletions must go through SetDeletedAt(clock.Now())
- Calling a.db directly inside helpers without TransactingRepo wrapping — falls off the ctx transaction
- Mixing namespace IS NULL and namespace IS NOT NULL in a single ListPrices query path
- Adding a source filter to ListOverrides (already constrained to source='manual')
- Editing openmeter/ent/db/ directly — regenerate via make generate after schema changes

## Decisions

- **Soft-delete for all price records** — Price history is needed for billing audits; hard deletes would destroy the audit trail for past invoice line resolutions
- **ResolvePrice skips TransactingRepo and reads directly** — It is intentionally read-only with no side-effects; the ORDER BY namespace DESC trick makes it a single-query hot path used in request path billing

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
