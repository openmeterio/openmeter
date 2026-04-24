# worker

<!-- archie:ai-start -->

> Organisational parent for charge-advancement workers. Splits the advance concern into two execution models: batch synchronous (advance/) for scheduled sweeps across all customers, and event-driven asynchronous (asyncadvance/) for single-customer advances triggered by a Watermill/event-bus message. Both children share the same constraint: all charge mutations must flow through charges.ChargeService, never through Ent adapters directly.

## Patterns

**Config struct + constructor validation** — Each child defines a Config struct with required fields and validates them in the constructor before returning the concrete type. advance/ validates inline in NewAdvancer; asyncadvance/ delegates to Config.Validate(). (`func NewAdvancer(config Config) (*AutoAdvancer, error) { if config.ChargesService == nil { return nil, fmt.Errorf(...) } }`)
**Service-only access — no direct Ent calls** — Both advance.AutoAdvancer and asyncadvance.Handler hold only a charges.ChargeService interface. Ent/adapter code is never imported or called in these packages. (`type AutoAdvancer struct { chargesService charges.ChargeService; logger *slog.Logger }`)
**Error accumulation in batch paths** — advance.All collects per-customer errors into []error and returns errors.Join — one failing customer never blocks others. asyncadvance.Handle returns the error directly (single-customer scope). (`var errs []error; for _, cust := range customers { if err := a.AdvanceCharges(ctx, cust); err != nil { errs = append(errs, ...) } }; return errors.Join(errs...)`)
**Pagination via CollectAll** — advance.ListCustomersToAdvance uses pagination.CollectAll with a pagination.NewPaginator closure so the full customer list is fetched page-by-page without manual cursor management. (`return pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[customer.CustomerID], error) { return a.chargesService.ListCustomersToAdvance(ctx, ...) }), defaultPageSize)`)
**Structured logging with slog** — Both packages use log/slog with context-aware methods (InfoContext, DebugContext, WarnContext, ErrorContext) and named key-value pairs for namespace and customer_id. (`h.logger.WarnContext(ctx, "failed to advance charges", slog.String("namespace", event.Namespace), slog.String("customer_id", event.CustomerID))`)
**Single-responsibility Handle method in async handler** — asyncadvance.Handler exposes exactly one public method: Handle(ctx, *charges.AdvanceChargesEvent) error. Batch/pagination logic must not be added here. (`func (h *Handler) Handle(ctx context.Context, event *charges.AdvanceChargesEvent) error { _, err := h.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID}}); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance/advance.go` | Batch sweeper: lists all customers with eligible charges via paginated service call, then advances each; accumulates errors across customers. | Time is captured once at the top of All() via time.Now() and passed to ListCustomersToAdvance — do not move it inside the paginator closure or per-customer calls. |
| `asyncadvance/asyncadvance.go` | Single-customer event handler wired as a Watermill message consumer; constructs customer.CustomerID from event fields and calls AdvanceCharges. | Returns the error from AdvanceCharges — do NOT return nil to suppress retries; Watermill uses the error for retry/dead-letter routing. |

## Anti-Patterns

- Calling Ent/adapter code directly — all mutations must go through charges.ChargeService
- Returning on first customer error in batch paths — must accumulate with errors.Join
- Introducing context.Background() or context.TODO() — always propagate caller's ctx
- Adding batch/pagination logic to asyncadvance.Handler — that belongs in advance.AutoAdvancer
- Niling out the error return in asyncadvance.Handle to silently swallow failures

## Decisions

- **Separate packages for batch (advance) and async (asyncadvance) advance paths** — Execution models differ — batch sweeps all customers with pagination and error accumulation; async handles a single Watermill event with pass-through error semantics. Mixing them would couple scheduling and event-bus concerns.
- **Error accumulation via errors.Join in the batch path instead of early return** — One failing customer must not prevent others from being advanced; all failures are surfaced together for observability.

## Example: Constructing and running the batch sweeper in a scheduled job

```
import (
	"context"

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
// ...
```

<!-- archie:ai-end -->
