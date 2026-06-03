# worker

<!-- archie:ai-start -->

> Organisational parent for charge-advancement workers split by execution model: advance/ for scheduled batch sweeps across all eligible customers (error accumulation + pagination) and asyncadvance/ for single-customer event-driven advancement triggered by a Watermill charges.AdvanceChargesEvent. Both children mutate only through charges.ChargeService, never Ent adapters directly.

## Patterns

**Config struct + constructor validation** — Each child defines a Config with required service fields and validates them in New*() before returning the concrete type; a nil ChargesService returns an error immediately. (`func NewAdvancer(config Config) (*AutoAdvancer, error) { if config.ChargesService == nil { return nil, fmt.Errorf("charges service is required") }; return &AutoAdvancer{chargesService: config.ChargesService, logger: config.Logger}, nil }`)
**Service-only access — no direct Ent calls** — advance.AutoAdvancer and asyncadvance.Handler hold only a charges.ChargeService interface; Ent/adapter packages are never imported here. (`type AutoAdvancer struct { chargesService charges.ChargeService; logger *slog.Logger }`)
**Error accumulation in batch, pass-through in async** — advance.All collects per-customer errors and returns errors.Join so one failure never blocks others; asyncadvance.Handle returns the error directly (single-customer scope) so Watermill can retry/DLQ. (`for _, cust := range customers { if err := a.AdvanceCharges(ctx, cust); err != nil { errs = append(errs, fmt.Errorf("customer %s: %w", cust.ID, err)) } }
return errors.Join(errs...)`)
**Pagination via pagination.CollectAll** — advance.ListCustomersToAdvance uses pagination.CollectAll with a pagination.NewPaginator closure to fetch the full customer list page-by-page without manual cursor management. (`pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[customer.CustomerID], error) { return a.chargesService.ListCustomersToAdvance(ctx, ...) }), defaultPageSize)`)
**Stable now captured once per batch** — Time is captured once at the top of All() and passed to ListCustomersToAdvance — never re-captured inside the paginator closure or per-customer calls. (`now := clock.Now(); customers, err := a.ListCustomersToAdvance(ctx, now)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance/advance.go` | Batch sweeper: paginated list of eligible customers, per-customer advance, error accumulation across customers. | time.Now() is captured once at the top of All() — do not capture inside the paginator closure or per-customer calls. |
| `asyncadvance/asyncadvance.go` | Single-customer Watermill handler: builds customer.CustomerID from event fields and calls AdvanceCharges. Exactly one public method Handle(ctx, *charges.AdvanceChargesEvent) error. | Return the AdvanceCharges error — never return nil to suppress retries; Watermill uses it for retry/DLQ routing. |

## Anti-Patterns

- Calling Ent/adapter code directly — all mutations must go through charges.ChargeService.
- Returning on first customer error in batch paths — must accumulate with errors.Join.
- Introducing context.Background()/context.TODO() — always propagate the caller's ctx.
- Adding batch/pagination logic to asyncadvance.Handler — that belongs in advance.AutoAdvancer.
- Returning nil from asyncadvance.Handle to swallow failures — Watermill uses the error for retry/DLQ routing.

## Decisions

- **Separate packages for batch (advance) and async (asyncadvance) advance paths.** — Execution models differ: batch sweeps all customers with pagination and error accumulation; async handles one Watermill event with pass-through error semantics. Mixing them couples scheduling and event-bus concerns.
- **Error accumulation via errors.Join in the batch path instead of early return.** — One failing customer must not prevent others from advancing; all failures surface together for observability and retry.

## Example: Construct and run the batch sweeper

```
import chargesworkeradvance "github.com/openmeterio/openmeter/openmeter/billing/charges/worker/advance"

advancer, err := chargesworkeradvance.NewAdvancer(chargesworkeradvance.Config{ChargesService: chargesSvc, Logger: logger})
if err != nil { return err }
if err := advancer.All(ctx, namespaces); err != nil { /* accumulated errors */ }
```

<!-- archie:ai-end -->
