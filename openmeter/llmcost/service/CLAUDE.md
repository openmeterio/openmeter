# service

<!-- archie:ai-start -->

> Thin service layer over llmcost.Adapter that adds the namespace-override overlay business logic on top of global prices, while delegating persistence. Implements llmcost.Service.

## Patterns

**Constructor injects adapter + logger** — New(adapter llmcost.Adapter, logger *slog.Logger) llmcost.Service stores both on the service struct. No DI wiring lives here. (`func New(adapter llmcost.Adapter, logger *slog.Logger) llmcost.Service { return &service{adapter: adapter, logger: logger} }`)
**Wrap mutations/reads in transaction.Run** — Service methods that compose multiple adapter calls run inside transaction.Run / transaction.RunWithNoValue against s.adapter, so overlay reads and writes share one tx. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[llmcost.Price], error) { ... })`)
**Namespace override overlay on ListPrices** — After fetching global prices, batch-fetch ListOverrides for the namespace, index them by (provider, model_id), and replace matching global rows in place. Skipped when namespace is empty, result empty, or the source filter excludes manual. (`overrideMap[overrideKey{string(o.Provider), o.ModelID}] = o; result.Items[i] = o`)
**Source-filter consistency guard** — sourceFilterExcludesManual inspects FilterString Eq/Ne to decide if applying the manual override overlay would violate a user source filter; if so, skip the overlay. (`if source.Eq != nil && *source.Eq != manual { return true }`)
**GetPrice override fallthrough** — When a global price is returned for a set namespace, ResolvePrice is consulted; a manual override replaces it, and IsGenericNotFoundError is treated as 'no override, keep global'. (`if models.IsGenericNotFoundError(err) { return price, nil }`)
**Pass-through for single-adapter operations** — ResolvePrice delegates directly without a transaction; CreateOverride/DeleteOverride/ListOverrides just wrap the single adapter call in transaction.Run for consistency. (`func (s *service) ResolvePrice(ctx, input) (llmcost.Price, error) { return s.adapter.ResolvePrice(ctx, input) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | service struct, New, all llmcost.Service methods, and the sourceFilterExcludesManual helper. | ListPrices and GetPrice mutate/replace result items with overrides — preserve the skip conditions (empty namespace, empty result, source filter excludes manual) or you leak overrides into filtered/global views. |
| `service_test.go` | Unit tests using an in-package mockAdapter + noopDriver covering the overlay matrix and sourceFilterExcludesManual truth table. | mockAdapter.Tx returns a noopDriver so transaction.Run works without a DB; new Service methods need a mockAdapter stub or tests won't compile. |

## Anti-Patterns

- Applying the manual override overlay when sourceFilterExcludesManual returns true (violates the caller's source filter).
- Overlaying overrides when input.Namespace is empty (global listing must stay global).
- Treating ResolvePrice not-found as a hard error in GetPrice instead of falling back to the global price.
- Putting Ent/DB queries in the service layer instead of delegating to llmcost.Adapter.
- Calling overlay logic outside transaction.Run so the override fetch and base read use different snapshots.

## Decisions

- **Overrides are overlaid in the service, not the adapter.** — The adapter keeps clean global vs namespace queries; the service expresses the 'namespace override wins' product rule and batches override lookups for O(1) replacement.
- **Source filter is honored by skipping the overlay rather than re-querying.** — Namespace overrides are always source=manual, so when the filter excludes manual the overlay would contradict the filter; skipping is cheaper and correct.

## Example: ListPrices with namespace override overlay guarded by the source filter

```
return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[llmcost.Price], error) {
	result, err := s.adapter.ListPrices(ctx, input)
	if err != nil { return pagination.Result[llmcost.Price]{}, err }
	if input.Namespace == "" || len(result.Items) == 0 || sourceFilterExcludesManual(input.Source) {
		return result, nil
	}
	overrides, err := s.adapter.ListOverrides(ctx, llmcost.ListOverridesInput{Namespace: input.Namespace, Provider: input.Provider, ModelID: input.ModelID})
	if err != nil { return pagination.Result[llmcost.Price]{}, err }
	overrideMap := make(map[overrideKey]llmcost.Price, len(overrides.Items))
	for _, o := range overrides.Items { overrideMap[overrideKey{string(o.Provider), o.ModelID}] = o }
	for i, p := range result.Items {
		if o, ok := overrideMap[overrideKey{string(p.Provider), p.ModelID}]; ok { result.Items[i] = o }
	}
	return result, nil
})
```

<!-- archie:ai-end -->
