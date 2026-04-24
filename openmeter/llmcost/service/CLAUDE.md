# service

<!-- archie:ai-start -->

> Business-logic layer implementing llmcost.Service: delegates persistence to llmcost.Adapter and adds namespace-override overlay logic for ListPrices and GetPrice without exposing it to callers.

## Patterns

**transaction.Run wrapping** — All methods that call the adapter wrap the body in transaction.Run or transaction.RunWithNoValue, providing consistent transaction boundaries at the service layer even though adapters also call TransactingRepo. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (T, error) { return s.adapter.SomeMethod(ctx, ...) })`)
**Namespace override overlay in ListPrices** — ListPrices fetches global prices, then batch-fetches overrides for the namespace and replaces matching entries in-memory using a provider+modelID map. Skipped entirely when namespace is empty or the source filter excludes manual prices. (`overrideMap[overrideKey{string(o.Provider), o.ModelID}] = o; result.Items[i] = o`)
**sourceFilterExcludesManual guard** — Before applying the overlay, sourceFilterExcludesManual(*filter.FilterString) checks Eq != 'manual' or Ne == 'manual'. If true the overlay is skipped to avoid returning manual items the caller filtered out. (`if sourceFilterExcludesManual(input.Source) { return result, nil }`)
**In-package unit tests with mock adapter** — Tests live in service_test.go (same package). They implement a mockAdapter satisfying llmcost.Adapter and a noopDriver for transaction.Driver. No real DB required. (`type mockAdapter struct { prices []llmcost.Price; overrides []llmcost.Price }; svc := New(adapter, slog.Default())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Concrete service implementation; contains overlay logic for ListPrices, GetPrice namespace promotion, and thin delegations for CreateOverride/DeleteOverride/ListOverrides/ResolvePrice | GetPrice only promotes to override when price.Namespace == nil (global) AND input.Namespace != '' — check both conditions before calling ResolvePrice again; returning the wrong price silently breaks billing |
| `service_test.go` | Comprehensive table-driven tests for overlay logic and sourceFilterExcludesManual | Tests use context.Background() because there is no testing.T available in helper functions — new tests should use t.Context() per project convention |

## Anti-Patterns

- Calling adapter methods directly without transaction.Run wrapping — service layer must own the transaction boundary
- Adding DB queries inside the overlay loop — batch-fetch overrides once before iterating
- Assuming source filter is a simple string — use sourceFilterExcludesManual helper, not inline string comparison

## Decisions

- **Overlay applied at service layer, not adapter layer** — The adapter exposes raw DB semantics; the service owns the rule 'namespace overrides take precedence', keeping adapter methods composable and testable independently

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
