# collect

<!-- archie:ai-start -->

> Batch invoice collection loop: lists GatheringInvoices whose NextCollectionAt is due, deduplicates by customer, and fans out parallel CollectCustomerInvoice calls that trigger billing.Service.InvoicePendingLines. Handles namespace locking, empty-line recalculation, and locked namespace exclusion.

## Patterns

**Config struct constructor with nil checks** — NewInvoiceCollector validates GatheringInvoiceService, BillingService, and Logger are non-nil before constructing the struct. (`func NewInvoiceCollector(config Config) (*InvoiceCollector, error)`)
**Input types with Validate() method** — ListCollectableInvoicesInput and CollectCustomerInvoiceInput each carry a Validate() error method checked at the top of their callers. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid input: %w", err) }`)
**Sentinel errors drive recovery, not failure** — ErrNamespaceLocked → warn + return nil. ErrInvoiceCreateNoLines → warn + RecalculateGatheringInvoices + return nil. Only unknown errors propagate. (`if errors.Is(err, billing.ErrNamespaceLocked) { a.logger.WarnContext(...); return nil, nil }`)
**Collection uses WithPartialInvoiceLinesDisabled option** — System-initiated collection must always call InvoicePendingLines with billing.WithPartialInvoiceLinesDisabled() to disable progressive billing. (`a.billingService.InvoicePendingLines(ctx, ..., billing.WithPartialInvoiceLinesDisabled())`)
**Locked namespace exclusion before fan-out** — All() filters customer IDs whose namespace appears in a.lockedNamespaces (slices.Contains) before building batches — prevents wasted goroutine launches. (`customerIDs = lo.Filter(..., func(id customer.CustomerID, _ int) bool { return !slices.Contains(a.lockedNamespaces, id.Namespace) })`)
**Batched parallel fan-out with errChan + sync.WaitGroup** — Same pattern as advance package: lo.Chunk, goroutines per batch item, buffered errChan sized to total customer count, sync.OnceFunc closer, errors.Join. (`errChan := make(chan error, len(customerIDs)); closeErrChan := sync.OnceFunc(func() { close(errChan) }); defer closeErrChan()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collect.go` | Entire package — InvoiceCollector with All(), ListCollectableInvoices(), CollectCustomerInvoice(), and NewInvoiceCollector(). GatheringInvoiceService and BillingService are held as separate fields for targeted use. | ListCollectableInvoices uses a filter.FilterTime OR clause to include invoices with nil NextCollectionAt — removing the nil branch would silently skip legacy invoices. errChan must be sized to len(customerIDs) not len(batches). |

## Anti-Patterns

- Calling InvoicePendingLines without billing.WithPartialInvoiceLinesDisabled() — allows progressive billing in system-initiated collection, which is explicitly forbidden
- Treating ErrNamespaceLocked or ErrInvoiceCreateNoLines as hard errors — both have defined recovery paths (skip and recalculate respectively)
- Skipping the locked namespace filter before fan-out — causes redundant errors for every customer in a locked namespace
- Using context.Background() inside goroutines instead of the parent ctx
- Forgetting lo.Uniq on customerIDs before batching — a customer with multiple gathering invoices would be collected twice

## Decisions

- **Collect at customer granularity, not invoice granularity** — InvoicePendingLines operates per-customer and may merge multiple pending lines into one invoice; iterating by invoice ID would cause double-collect races.
- **ErrInvoiceCreateNoLines triggers RecalculateGatheringInvoices instead of failing** — This state indicates stale gathering invoice state (possible data inconsistency); a recalculation heals it without surfacing an error to the caller.

## Example: Collect all due invoices for a namespace in batches of 100

```
collector, _ := billingworkercollect.NewInvoiceCollector(billingworkercollect.Config{
    GatheringInvoiceService: billingSvc,
    BillingService:          billingSvc,
    Logger:                  logger,
    LockedNamespaces:        lockedNS,
})
if err := collector.All(ctx, []string{"default"}, nil, 100); err != nil {
    return err
}
```

<!-- archie:ai-end -->
