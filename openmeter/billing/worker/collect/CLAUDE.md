# collect

<!-- archie:ai-start -->

> Batch invoice collection loop: lists GatheringInvoices whose NextCollectionAt is due (including nil-NextCollectionAt legacy invoices), deduplicates by customer, filters locked namespaces, then fans out parallel CollectCustomerInvoice calls that trigger billing.Service.InvoicePendingLines with partial-lines disabled.

## Patterns

**Config struct constructor with nil checks** — NewInvoiceCollector validates GatheringInvoiceService, BillingService, and Logger are non-nil before constructing. LockedNamespaces is optional. (`func NewInvoiceCollector(config Config) (*InvoiceCollector, error) { if config.GatheringInvoiceService == nil { return nil, fmt.Errorf(...) } }`)
**Input types carry Validate() method checked at caller entry** — ListCollectableInvoicesInput and CollectCustomerInvoiceInput each have a Validate() error method checked at the top of their callers before any service calls. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid input: %w", err) }`)
**Sentinel errors drive recovery not failure** — ErrNamespaceLocked → warn + return nil. ErrInvoiceCreateNoLines → warn + RecalculateGatheringInvoices + return nil. Only unknown errors propagate. (`if errors.Is(err, billing.ErrNamespaceLocked) { a.logger.WarnContext(ctx, "namespace is locked, skipping", ...); return nil, nil }`)
**InvoicePendingLines always called with WithPartialInvoiceLinesDisabled** — System-initiated collection must always use billing.WithPartialInvoiceLinesDisabled() to prevent progressive billing during automated collection. (`a.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{Customer: params.CustomerID}, billing.WithPartialInvoiceLinesDisabled())`)
**Locked namespace exclusion before fan-out** — All() filters customer IDs whose namespace appears in a.lockedNamespaces using slices.Contains before building batches — prevents wasted goroutine launches against locked namespaces. (`customerIDs = lo.Filter(lo.Uniq(customerIDs), func(id customer.CustomerID, _ int) bool { return !slices.Contains(a.lockedNamespaces, id.Namespace) })`)
**Batched parallel fan-out with errChan + sync.WaitGroup (same as advance package)** — lo.Chunk splits by batchSize; goroutines per batch item write to a buffered errChan sized to total customer count; sync.OnceFunc closes channel; errors.Join collects. (`errChan := make(chan error, len(customerIDs)); closeErrChan := sync.OnceFunc(func() { close(errChan) }); defer closeErrChan()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collect.go` | Entire package — InvoiceCollector with All(), ListCollectableInvoices(), CollectCustomerInvoice(), NewInvoiceCollector(); input types ListCollectableInvoicesInput and CollectCustomerInvoiceInput with Validate(). | ListCollectableInvoices uses a filter.FilterTime OR clause to include invoices with nil NextCollectionAt — removing the nil branch silently skips legacy invoices. errChan must be sized to len(customerIDs) not len(batches). lo.Uniq must be applied to customerIDs before batching or a customer with multiple gathering invoices gets collected twice. |

## Anti-Patterns

- Calling InvoicePendingLines without billing.WithPartialInvoiceLinesDisabled() — allows progressive billing in system-initiated collection, which is explicitly forbidden
- Treating ErrNamespaceLocked or ErrInvoiceCreateNoLines as hard errors — both have defined recovery paths
- Skipping the locked namespace filter before fan-out — causes redundant errors for every customer in a locked namespace
- Using context.Background() inside goroutines instead of the parent ctx
- Forgetting lo.Uniq on customerIDs before batching — a customer with multiple gathering invoices gets collected twice

## Decisions

- **Collect at customer granularity, not invoice granularity** — InvoicePendingLines operates per-customer and may merge multiple pending lines into one invoice; iterating by invoice ID would cause double-collect races for customers with multiple gathering invoices.
- **ErrInvoiceCreateNoLines triggers RecalculateGatheringInvoices instead of failing** — This state indicates stale gathering invoice state (possible data inconsistency); a recalculation heals it without surfacing an error to the caller or causing unnecessary alert noise.

## Example: Collect all due gathering invoices for a namespace in batches of 100

```
import billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"

collector, err := billingworkercollect.NewInvoiceCollector(billingworkercollect.Config{
    GatheringInvoiceService: billingSvc,
    BillingService:          billingSvc,
    Logger:                  logger,
    LockedNamespaces:        lockedNS,
})
if err != nil {
    return err
}
return collector.All(ctx, []string{"default"}, nil, 100)
```

<!-- archie:ai-end -->
