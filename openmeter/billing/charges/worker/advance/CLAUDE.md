# advance

<!-- archie:ai-start -->

> Batch driver that auto-advances all eligible usage-based charges across customers. AutoAdvancer.All scans namespaces for customers with charges past their advance-after watermark and advances each, isolating per-customer failures. It is a thin orchestration layer over charges.ChargeService — it holds no business logic of its own.

## Patterns

**Config + validating constructor** — Dependencies arrive via a Config struct; NewAdvancer returns (*AutoAdvancer, error) and rejects nil ChargesService or Logger with fmt.Errorf instead of panicking. (`NewAdvancer(Config{ChargesService: svc, Logger: log})`)
**Per-customer error isolation** — All() collects per-customer failures into var errs []error and returns errors.Join(errs...) so one customer's failure never aborts the batch; each failure is also logged at ErrorContext. (`errs = append(errs, fmt.Errorf("... [namespace=%s customer=%s]: %w", ...)); return errors.Join(errs...)`)
**Paginate via pagination.CollectAll** — ListCustomersToAdvance wraps chargesService.ListCustomersToAdvance in pagination.NewPaginator + CollectAll with defaultPageSize (10_000); never hand-roll page loops. (`pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx, page) (...) { return a.chargesService.ListCustomersToAdvance(...) }), defaultPageSize)`)
**Context-aware structured logging** — All logging uses *Context variants (InfoContext/DebugContext/WarnContext/ErrorContext) with slog.String attrs for namespace and customer_id. (`a.logger.ErrorContext(ctx, "failed to auto-advance charges", slog.String("namespace", cust.Namespace), ...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Defines AutoAdvancer (All, ListCustomersToAdvance, AdvanceCharges), Config, and NewAdvancer. | AdvanceAfterLTE uses time.Now() captured once at the start of ListCustomersToAdvance — the watermark is the scan-start instant, not per-customer. AdvanceCharges discards the AdvanceCharges result value (only the error). |

## Anti-Patterns

- Returning early from All() on the first customer error instead of accumulating into errors.Join — this would skip remaining customers.
- Adding charge advancement business logic here; this package only orchestrates charges.ChargeService.
- Falling back to slog.Default() instead of requiring Logger via Config (constructor rejects nil).

## Decisions

- **Best-effort batch with error aggregation rather than fail-fast.** — Auto-advance is a sweep across all customers; one bad customer must not block billing progress for the rest.

## Example: Run auto-advance across namespaces, isolating per-customer failures

```
func (a *AutoAdvancer) All(ctx context.Context, namespaces []string) error {
	customers, err := a.ListCustomersToAdvance(ctx, namespaces)
	if err != nil {
		return fmt.Errorf("failed to list customers to advance charges: %w", err)
	}
	var errs []error
	for _, cust := range customers {
		if err := a.AdvanceCharges(ctx, cust); err != nil {
			errs = append(errs, fmt.Errorf("failed to auto-advance charges [namespace=%s customer=%s]: %w", cust.Namespace, cust.ID, err))
		}
	}
	return errors.Join(errs...)
}
```

<!-- archie:ai-end -->
