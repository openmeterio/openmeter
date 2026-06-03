# collect

<!-- archie:ai-start -->

> Batch invoice collection loop: lists GatheringInvoices whose NextCollectionAt is due (including nil-NextCollectionAt legacy invoices), deduplicates by customer, filters locked namespaces, then fans out parallel CollectCustomerInvoice calls triggering billing.Service.InvoicePendingLines with partial lines disabled.

## Patterns

**Config struct constructor with nil checks** — NewInvoiceCollector validates GatheringInvoiceService, BillingService, and Logger non-nil; LockedNamespaces is optional. (`func NewInvoiceCollector(config Config) (*InvoiceCollector, error) { if config.GatheringInvoiceService == nil { return nil, fmt.Errorf(...) } }`)
**Input types carry Validate() checked at entry** — ListCollectableInvoicesInput and CollectCustomerInvoiceInput each have Validate() checked at the top of their callers before any service calls. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid input: %w", err) }`)
**Sentinel errors drive recovery not failure** — ErrNamespaceLocked -> warn + nil; ErrInvoiceCreateNoLines -> warn + RecalculateGatheringInvoices + nil. Only unknown errors propagate. (`if errors.Is(err, billing.ErrNamespaceLocked) { a.logger.WarnContext(ctx, "namespace is locked, skipping", ...); return nil, nil }`)
**InvoicePendingLines always with PartialInvoiceLinesDisabled** — System-initiated collection always uses billing.WithPartialInvoiceLinesDisabled() to prevent progressive billing during automated collection. (`a.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{Customer: params.CustomerID}, billing.WithPartialInvoiceLinesDisabled())`)
**Locked namespace exclusion before fan-out** — All() filters customer IDs whose namespace is in a.lockedNamespaces via slices.Contains before building batches. (`customerIDs = lo.Filter(lo.Uniq(customerIDs), func(id customer.CustomerID, _ int) bool { return !slices.Contains(a.lockedNamespaces, id.Namespace) })`)
**Batched parallel fan-out (same as advance)** — lo.Chunk by batchSize; goroutines per item write to a buffered errChan sized to total customer count; sync.OnceFunc closes; errors.Join collects. (`errChan := make(chan error, len(customerIDs)); closeErrChan := sync.OnceFunc(func() { close(errChan) }); defer closeErrChan()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collect.go` | Entire package — InvoiceCollector with All(), ListCollectableInvoices(), CollectCustomerInvoice(), NewInvoiceCollector(); input types with Validate(). | ListCollectableInvoices uses a filter.FilterTime OR clause to include nil NextCollectionAt — removing the nil branch silently skips legacy invoices; errChan must be sized to len(customerIDs); lo.Uniq must run on customerIDs before batching. |

## Anti-Patterns

- Calling InvoicePendingLines without billing.WithPartialInvoiceLinesDisabled() — allows forbidden progressive billing in system collection.
- Treating ErrNamespaceLocked or ErrInvoiceCreateNoLines as hard errors — both have recovery paths.
- Skipping the locked-namespace filter before fan-out — redundant errors for every customer in a locked namespace.
- Using context.Background() inside goroutines instead of the parent ctx.
- Forgetting lo.Uniq on customerIDs before batching — a customer with multiple gathering invoices gets collected twice.

## Decisions

- **Collect at customer granularity, not invoice granularity.** — InvoicePendingLines operates per-customer and may merge multiple pending lines into one invoice; iterating by invoice ID would cause double-collect races.
- **ErrInvoiceCreateNoLines triggers RecalculateGatheringInvoices instead of failing.** — This state indicates stale gathering-invoice state; a recalculation heals it without surfacing an error or alert noise.

## Example: Collect all due gathering invoices in batches of 100

```
import billingworkercollect "github.com/openmeterio/openmeter/openmeter/billing/worker/collect"

collector, err := billingworkercollect.NewInvoiceCollector(billingworkercollect.Config{GatheringInvoiceService: billingSvc, BillingService: billingSvc, Logger: logger, LockedNamespaces: lockedNS})
if err != nil { return err }
return collector.All(ctx, []string{"default"}, nil, 100)
```

<!-- archie:ai-end -->
