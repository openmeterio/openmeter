# service

<!-- archie:ai-start -->

> Business-logic layer implementing llmcost.Service: delegates persistence to llmcost.Adapter and adds namespace-override overlay logic for ListPrices and GetPrice without exposing raw DB semantics to callers. Owns the rule 'namespace overrides take precedence over global prices'.

## Patterns

**transaction.Run wrapping at service boundary** — All methods that call the adapter wrap the body in transaction.Run or transaction.RunWithNoValue, providing consistent transaction boundaries at the service layer even though adapters also call TransactingRepo internally. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[llmcost.Price], error) { return s.adapter.ListPrices(ctx, input) })`)
**Namespace override overlay in ListPrices** — ListPrices fetches global prices, batch-fetches overrides for the namespace, then replaces matching entries in-memory using a provider+modelID map. Skipped when namespace is empty, result is empty, or the source filter excludes manual prices. (`overrideMap[overrideKey{string(o.Provider), o.ModelID}] = o; for i, p := range result.Items { if o, ok := overrideMap[...]; ok { result.Items[i] = o } }`)
**sourceFilterExcludesManual guard before overlay** — Before applying the overlay, call sourceFilterExcludesManual(*filter.FilterString) which checks Eq != 'manual' or Ne == 'manual'. Use this helper, not inline string comparison. (`if sourceFilterExcludesManual(input.Source) { return result, nil }`)
**Single batch-fetch of overrides before the loop** — Batch-fetch all namespace overrides once before iterating the result slice. Never add DB queries inside the overlay loop. (`overrides, err := s.adapter.ListOverrides(ctx, llmcost.ListOverridesInput{Namespace: input.Namespace, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Concrete service implementation. Contains overlay logic for ListPrices, GetPrice namespace promotion, and thin delegations for CreateOverride/DeleteOverride/ListOverrides/ResolvePrice. | GetPrice promotes to override only when price.Namespace == nil (global) AND input.Namespace != '' — both conditions must be checked. ResolvePrice delegates directly to the adapter without transaction.Run (intentional: read-only hot path). |
| `service_test.go` | Table-driven unit tests for overlay logic using a mockAdapter; no real DB required. | Tests use context.Background() in helpers where *testing.T is unavailable — new tests added with a *testing.T receiver should use t.Context() per project convention. |

## Anti-Patterns

- Calling adapter methods directly without transaction.Run wrapping — service layer must own the transaction boundary
- Adding DB queries inside the per-item overlay loop — batch-fetch overrides once before iterating
- Inline string comparison for the source filter instead of using sourceFilterExcludesManual helper
- Promoting a price in GetPrice when price.Namespace is already set (override found by adapter directly) — only promote when price.Namespace == nil

## Decisions

- **Overlay applied at service layer, not adapter layer** — The adapter exposes raw DB semantics; the service owns the precedence rule. This keeps adapter methods independently composable and testable.

## Example: Add a new service method that delegates to adapter with a transaction

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
