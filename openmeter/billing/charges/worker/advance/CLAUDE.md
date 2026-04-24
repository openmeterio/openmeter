# advance

<!-- archie:ai-start -->

> Batch auto-advance worker that iterates all customers with eligible charges and calls charges.Service.AdvanceCharges for each; errors are collected and joined rather than short-circuiting, so one failing customer does not block others.

## Patterns

**Fan-out with error accumulation** — Loop over all customers, accumulate errors via `errs = append(errs, ...)`, return `errors.Join(errs...)` so all failures are reported and processing continues. (`for _, cust := range customers { if err := a.AdvanceCharges(ctx, cust); err != nil { errs = append(errs, err) } }; return errors.Join(errs...)`)
**Pagination via CollectAll** — Use `pagination.CollectAll` with a `pagination.NewPaginator` closure to fetch all pages of customers without manual page-loop code. (`pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[customer.CustomerID], error) { return a.chargesService.ListCustomersToAdvance(ctx, ...) }), defaultPageSize)`)
**Config struct + constructor validation** — All dependencies passed via a `Config` struct; `NewAdvancer` validates non-nil fields before constructing the struct. (`func NewAdvancer(config Config) (*AutoAdvancer, error) { if config.ChargesService == nil { return nil, fmt.Errorf(...) } ... }`)
**Advance via service interface only** — All charge advancement is driven through `charges.ChargeService.AdvanceCharges` — never call adapters or Ent directly from this layer. (`a.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customerID})`)
**Structured logging with slog** — Use `slog.InfoContext`/`DebugContext`/`ErrorContext`/`WarnContext` with named key-value pairs (`slog.String`) for all log statements. (`a.logger.ErrorContext(ctx, "failed to auto-advance charges", slog.String("namespace", ...), slog.String("customer_id", ...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Sole file; defines AutoAdvancer struct, All/ListCustomersToAdvance/AdvanceCharges methods, Config struct, and NewAdvancer constructor. | defaultPageSize=10_000 caps single-page fetches; `now` is captured once at the start of ListCustomersToAdvance to give a stable cutoff for the entire batch run. |

## Anti-Patterns

- Returning on first customer error — must accumulate and join errors
- Calling Ent/adapter code directly instead of going through charges.ChargeService
- Introducing context.Background() — always propagate the caller's ctx
- Hard-coding namespaces instead of accepting them as a parameter

## Decisions

- **Error accumulation instead of early return** — A single customer failure should not prevent advancing charges for all other customers in the batch.
- **Time captured once per All() call** — Using a single `now` for the entire batch ensures consistent cutoff semantics; per-customer clock calls could admit races where newly eligible charges slip into or out of the window mid-run.

## Example: Full batch advance entry point

```
func (a *AutoAdvancer) All(ctx context.Context, namespaces []string) error {
	customers, err := a.ListCustomersToAdvance(ctx, namespaces)
	if err != nil {
		return fmt.Errorf("failed to list customers to advance charges: %w", err)
	}
	var errs []error
	for _, cust := range customers {
		if err := a.AdvanceCharges(ctx, cust); err != nil {
			errs = append(errs, fmt.Errorf("[namespace=%s customer=%s]: %w", cust.Namespace, cust.ID, err))
		}
	}
	return errors.Join(errs...)
}
```

<!-- archie:ai-end -->
