# service

<!-- archie:ai-start -->

> Business-logic layer implementing llmcost.Service: delegates persistence to llmcost.Adapter and adds the namespace-override overlay for ListPrices and GetPrice without exposing raw DB semantics. Owns the rule 'namespace overrides take precedence over global prices'.

## Patterns

**transaction.Run wrapping at the service boundary** — Methods calling the adapter wrap the body in transaction.Run / RunWithNoValue, providing consistent transaction boundaries at the service layer even though adapters also call TransactingRepo internally. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[llmcost.Price], error) { return s.adapter.ListPrices(ctx, input) })`)
**Namespace override overlay in ListPrices** — Fetch global prices, batch-fetch namespace overrides, then replace matching entries in-memory via a provider+modelID map. Skipped when namespace is empty, result is empty, or the source filter excludes manual. (`overrideMap[overrideKey{string(o.Provider), o.ModelID}] = o; for i, p := range result.Items { if o, ok := overrideMap[...]; ok { result.Items[i] = o } }`)
**sourceFilterExcludesManual guard before overlay** — Call sourceFilterExcludesManual(*filter.FilterString) (checks Eq != 'manual' or Ne == 'manual') before applying the overlay; use the helper, not inline comparison. (`if sourceFilterExcludesManual(input.Source) { return result, nil }`)
**Single batch-fetch of overrides before the loop** — Fetch all namespace overrides once before iterating the result slice; never query inside the overlay loop. (`overrides, err := s.adapter.ListOverrides(ctx, llmcost.ListOverridesInput{Namespace: input.Namespace})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Concrete service: overlay logic for ListPrices, GetPrice namespace promotion, and thin delegations for CreateOverride/DeleteOverride/ListOverrides/ResolvePrice. | GetPrice promotes to override only when price.Namespace == nil (global) AND input.Namespace != '' — check both. ResolvePrice delegates directly without transaction.Run (read-only hot path). |
| `service_test.go` | Table-driven unit tests of overlay logic using a mockAdapter; no real DB. | Existing helpers use context.Background() where no *testing.T is available; new *testing.T tests should use t.Context(). |

## Anti-Patterns

- Calling adapter methods directly without transaction.Run wrapping — the service owns the transaction boundary.
- Adding DB queries inside the per-item overlay loop instead of batch-fetching once.
- Inline string comparison for the source filter instead of sourceFilterExcludesManual.
- Promoting a price in GetPrice when price.Namespace is already set — only promote when price.Namespace == nil.

## Decisions

- **Override overlay applied at the service layer, not the adapter.** — The adapter exposes raw DB semantics; the service owns the precedence rule, keeping adapter methods independently composable and testable.

## Example: A service method delegating to the adapter within a transaction

```
func (s *service) BulkDelete(ctx context.Context, ids []string) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		for _, id := range ids {
			if err := s.adapter.DeleteOverride(ctx, llmcost.DeleteOverrideInput{ID: id}); err != nil {
				return err
			}
		}
		return nil
	})
}
```

<!-- archie:ai-end -->
