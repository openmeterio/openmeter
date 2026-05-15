# advance

<!-- archie:ai-start -->

> Batch auto-advance worker that iterates all customers with eligible charges and calls charges.Service.AdvanceCharges for each; errors are accumulated and joined so one failing customer does not block others.

## Patterns

**Fan-out with error accumulation** — Loop over all customers, append errors to a slice, return errors.Join(errs...) — never return on first failure. (`var errs []error
for _, cust := range customers {
    if err := a.AdvanceCharges(ctx, cust); err != nil {
        errs = append(errs, fmt.Errorf("[namespace=%s customer=%s]: %w", cust.Namespace, cust.ID, err))
    }
}
return errors.Join(errs...)`)
**Pagination via CollectAll** — Use pagination.CollectAll with pagination.NewPaginator to page through all eligible customers without a manual page loop. (`pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[customer.CustomerID], error) {
    return a.chargesService.ListCustomersToAdvance(ctx, charges.ListCustomersToAdvanceInput{Page: page, Namespaces: namespaces, AdvanceAfterLTE: now})
}), defaultPageSize)`)
**Config struct + constructor validation** — All dependencies passed via a Config struct; NewAdvancer validates non-nil fields before constructing the struct. (`func NewAdvancer(config Config) (*AutoAdvancer, error) {
    if config.ChargesService == nil { return nil, fmt.Errorf("charges service is required") }
    if config.Logger == nil { return nil, fmt.Errorf("logger is required") }
    return &AutoAdvancer{chargesService: config.ChargesService, logger: config.Logger}, nil
}`)
**Advance via service interface only** — All charge advancement is driven through charges.ChargeService.AdvanceCharges — never call adapters or Ent directly. (`a.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customerID})`)
**Stable now captured once per batch** — Capture time.Now() once at the start of ListCustomersToAdvance and pass it as AdvanceAfterLTE to the entire batch, preventing per-customer clock races. (`now := time.Now()
// pass now to ListCustomersToAdvance, not time.Now() inside the paginator closure`)
**Structured logging with slog** — Use slog.InfoContext/DebugContext/ErrorContext/WarnContext with named slog.String key-value pairs for all log statements. (`a.logger.ErrorContext(ctx, "failed to auto-advance charges",
    slog.String("namespace", cust.Namespace),
    slog.String("customer_id", cust.ID),
    slog.String("error", err.Error()),
)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Sole file; defines AutoAdvancer struct, All/ListCustomersToAdvance/AdvanceCharges methods, Config struct, and NewAdvancer constructor. | defaultPageSize=10_000 caps single-page fetches; `now` is captured once at the start of ListCustomersToAdvance to give a stable cutoff for the entire batch run. |

## Anti-Patterns

- Returning on first customer error — must accumulate and join all errors
- Calling Ent/adapter code directly instead of going through charges.ChargeService
- Introducing context.Background() — always propagate the caller's ctx
- Hard-coding namespaces instead of accepting them as a parameter
- Capturing time.Now() inside the paginator closure rather than once before it

## Decisions

- **Error accumulation instead of early return** — A single customer failure should not prevent advancing charges for all other customers in the batch; errors.Join collects all failures for a complete failure report.
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
