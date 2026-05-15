# worker

<!-- archie:ai-start -->

> Organisational parent for charge-advancement workers split by execution model: advance/ for scheduled batch sweeps across all customers with error accumulation and pagination, and asyncadvance/ for single-customer event-driven advancement triggered by a Watermill AdvanceChargesEvent. Both children share the same constraint: all mutations flow through charges.ChargeService, never through Ent adapters directly.

## Patterns

**Config struct + constructor validation before use** — Each child defines a Config struct with required service fields and validates them in New*() before returning the concrete type. Missing required fields (e.g. ChargesService == nil) return an error immediately. (`func NewAdvancer(config Config) (*AutoAdvancer, error) {
    if config.ChargesService == nil { return nil, fmt.Errorf("charges service is required") }
    return &AutoAdvancer{chargesService: config.ChargesService, logger: config.Logger}, nil
}`)
**Service-only access — no direct Ent calls** — Both advance.AutoAdvancer and asyncadvance.Handler hold only a charges.ChargeService interface. Ent/adapter packages are never imported or called in these packages. (`type AutoAdvancer struct { chargesService charges.ChargeService; logger *slog.Logger }`)
**Error accumulation via errors.Join in batch paths** — advance.All collects per-customer errors into []error and returns errors.Join — one failing customer never blocks others. asyncadvance.Handle returns the error directly (single-customer scope). (`var errs []error
for _, cust := range customers {
    if err := a.AdvanceCharges(ctx, cust); err != nil {
        errs = append(errs, fmt.Errorf("customer %s: %w", cust.ID, err))
    }
}
return errors.Join(errs...)`)
**Pagination via pagination.CollectAll with NewPaginator** — advance.ListCustomersToAdvance uses pagination.CollectAll with a pagination.NewPaginator closure so the full customer list is fetched page-by-page without manual cursor management. (`return pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[customer.CustomerID], error) {
    return a.chargesService.ListCustomersToAdvance(ctx, ...)
}), defaultPageSize)`)
**Structured logging with slog using context-aware methods** — Both packages use log/slog with InfoContext/DebugContext/WarnContext/ErrorContext and named key-value pairs for namespace and customer_id. (`h.logger.WarnContext(ctx, "failed to advance charges", slog.String("namespace", event.Namespace), slog.String("customer_id", event.CustomerID))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance/advance.go` | Batch sweeper: lists all customers with eligible charges via paginated service call, advances each, accumulates errors across customers. | time.Now() is captured once at the top of All() and passed to ListCustomersToAdvance — do not capture it inside the paginator closure or per-customer calls. |
| `asyncadvance/asyncadvance.go` | Single-customer Watermill event handler: constructs customer.CustomerID from event fields and calls AdvanceCharges. Exposes exactly one public method: Handle(ctx, *charges.AdvanceChargesEvent) error. | Returns the error from AdvanceCharges — do NOT return nil to suppress retries; Watermill uses the error for retry/dead-letter routing. |

## Anti-Patterns

- Calling Ent/adapter code directly — all mutations must go through charges.ChargeService.
- Returning on first customer error in batch paths — must accumulate with errors.Join.
- Introducing context.Background() or context.TODO() — always propagate caller's ctx.
- Adding batch/pagination logic to asyncadvance.Handler — that belongs in advance.AutoAdvancer.
- Returning nil from asyncadvance.Handle to silently swallow failures — Watermill uses the error for retry/DLQ routing.

## Decisions

- **Separate packages for batch (advance) and async (asyncadvance) advance paths** — Execution models differ: batch sweeps all customers with pagination and error accumulation; async handles a single Watermill event with pass-through error semantics. Mixing them would couple scheduling and event-bus concerns.
- **Error accumulation via errors.Join in the batch path instead of early return** — One failing customer must not prevent others from being advanced; all failures are surfaced together for observability and retry.

## Example: Constructing and running the batch sweeper

```
import (
    chargesworkeradvance "github.com/openmeterio/openmeter/openmeter/billing/charges/worker/advance"
)

advancer, err := chargesworkeradvance.NewAdvancer(chargesworkeradvance.Config{
    ChargesService: chargesSvc,
    Logger:         logger,
})
if err != nil {
    return err
}
// Sweep all namespaces; errors from individual customers are accumulated, not short-circuited
if err := advancer.All(ctx, namespaces); err != nil {
    // log or surface accumulated errors
}
```

<!-- archie:ai-end -->
